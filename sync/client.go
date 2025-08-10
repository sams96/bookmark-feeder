package sync

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"ffsyncclient/cli"
	"ffsyncclient/models"
	"ffsyncclient/syncclient"
)

type SyncClient struct {
	ffsCtx      *cli.FFSContext
	client      *syncclient.FxAClient
	sessionPath string
	session     *syncclient.FFSyncSession
}

func NewSyncClient(sessionPath string) (*SyncClient, error) {
	opts := cli.Options{
		AuthServerURL:   "https://api.accounts.firefox.com/v1",
		TokenServerURL:  "https://token.services.mozilla.com",
		SessionFilePath: sessionPath,
		TimeZone:        time.UTC,
	}

	ffsCtx, err := cli.NewContext(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create new FFS context: %w", err)
	}

	client := syncclient.NewFxAClient(ffsCtx, opts.AuthServerURL)

	return &SyncClient{
		ffsCtx: ffsCtx,
		client: client,
	}, nil
}

func (sc *SyncClient) Login(email string, password string) error {
	const maxRetries = 10
	const retryDelay = 30 * time.Second

	// TODO: configurable?
	const deviceName = "sync-client"
	const deviceType = "cli"

	for i := range maxRetries {
		slog.Debug("Attempting to log in...", "attempt", i+1)
		loginSession, verificationMethod, err := sc.client.Login(sc.ffsCtx, email, password)
		if err != nil {
			if strings.Contains(err.Error(), "ffsync.direct_out") {
				slog.Warn("Login requires email verification", "delay", retryDelay)
				fmt.Printf("An error occurred: You must verify this login attempt via an email sent to your address.\n")
				fmt.Printf("Please check your email and approve the login. Retrying in %d seconds...\n", int(retryDelay.Seconds()))
				time.Sleep(retryDelay)
				continue
			}
			return err
		}

		if verificationMethod == syncclient.VerificationTOTP2FA || verificationMethod == syncclient.VerificationMail2FA {
			slog.Info("2FA required, prompting user for OTP")
			fmt.Println("Enter your OTP (2-Factor Authentication Code): ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			otp := strings.TrimSpace(input)

			slog.Debug("Verifying with OTP", "otp", otp)
			err := sc.client.VerifyWithOTP(sc.ffsCtx, loginSession, otp)
			if err != nil {
				return err
			}
		}

		slog.Info("Registering device", "deviceName", deviceName)
		err = sc.client.RegisterDevice(sc.ffsCtx, loginSession, deviceName, deviceType)
		if err != nil {
			return err
		}

		slog.Info("Fetching session keys")
		keyA, keyB, err := sc.client.FetchKeys(sc.ffsCtx, loginSession)
		if err != nil {
			return err
		}
		keyedSession := loginSession.Extend(keyA, keyB)

		slog.Info("Acquiring OAuth Token")
		oauthSession, err := sc.client.AcquireOAuthToken(sc.ffsCtx, keyedSession)
		if err != nil {
			return err
		}

		slog.Info("Getting HAWK Credentials")
		hawkSession, err := sc.client.HawkAuth(sc.ffsCtx, oauthSession)
		if err != nil {
			return err
		}

		slog.Info("Getting Crypto Keys")
		cryptoSession, err := sc.client.GetCryptoKeys(sc.ffsCtx, hawkSession)
		if err != nil {
			return err
		}

		slog.Info("Login successful")
		syncSession := cryptoSession.Reduce()

		slog.Info("Saving session to file", "path", sc.ffsCtx.Opt.SessionFilePath)
		err = syncSession.Save(sc.ffsCtx.Opt.SessionFilePath)
		if err != nil {
			slog.Error("Error saving session", "error", err)
			os.Exit(1)
		}
		slog.Info("Session saved successfully.")

		return nil
	}

	return fmt.Errorf("failed to verify login after %d attempts", maxRetries)
}

func (sc *SyncClient) GetBookmarks() ([]models.BookmarkRecord, error) {
	session, err := syncclient.LoadSession(sc.ffsCtx, sc.ffsCtx.Opt.SessionFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	session, _, err = sc.client.RefreshSession(sc.ffsCtx, session, false)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh session: %w", err)
	}

	records, err := sc.client.ListRecords(sc.ffsCtx, session, "bookmarks", nil, nil, false, true, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmarks: %w", err)
	}
	slog.Info("Bookmarks fetched successfully", "count", len(records))

	bookmarks, err := models.UnmarshalBookmarks(sc.ffsCtx, records, false)
	if err != nil {
		slog.Error("An error occurred while unmarshalling bookmarks", "error", err)
		os.Exit(1)
	}

	return bookmarks, nil
}

func FilterBookmarks(bookmarks []models.BookmarkRecord, FolderName string) []models.BookmarkRecord {
	var ret []models.BookmarkRecord
	for _, bookmark := range bookmarks {
		if bookmark.ParentName == FolderName {
			ret = append(ret, bookmark)
		}
	}

	return ret
}
