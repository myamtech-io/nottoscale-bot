package simulationcraft

import (
	"encoding/base64"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"time"
)

// Variables used for command line parameters
var (
	execLocation     string
	bnetClientID     string
	bnetAccessToken  string
	bnetClientSecret string
	client           *http.Client
)

func init() {
	tr := &http.Transport{
		MaxIdleConns:       1,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client = &http.Client{Transport: tr}
}

// Initialize foo
func Initialize(simcPath string, providedBnetClientID string, providedBnetClientSecret string) error {
	execLocation = path.Join(simcPath, "simc")

	bnetClientID = providedBnetClientID
	bnetClientSecret = providedBnetClientSecret

	accessToken, err := getNewAccessToken()

	if err != nil {
		return err
	}

	err = setAccessTokenForSimcraft(accessToken)

	if err != nil {
		return err
	}

	go refreshAccessToken(accessToken)

	bnetAccessToken = accessToken.AccessToken

	return nil
}

// Simulate d
func Simulate(realm string, name string, s *discordgo.Session, m *discordgo.MessageCreate) {
	folder := b64.StdEncoding.EncodeToString([]byte(realm + name))
	dir, err := ioutil.TempDir("", folder)

	if err != nil {
		log.Error("Could not create simulation directory: ", err)
		return
	}

	destinationFile := path.Join(dir, name+".html")

	var logger = log.WithFields(log.Fields{
		"name":       name,
		"realm":      realm,
		"output":     destinationFile,
		"executable": execLocation,
	})

	s.ChannelMessageSend(m.ChannelID, "Running simulation...")

	// Remove the entire directory at the end of this call
	defer os.RemoveAll(dir)

	cmd := exec.Command(execLocation, "armory=us,"+realm+","+name, "html="+destinationFile)
	logger.Info("Running command and waiting for it to finish... Writing to ")

	stdoutStderr, err := cmd.CombinedOutput()

	logger.Info("Simulation finished running")

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Errored: "+string(stdoutStderr))

		logger.Error("Received error while running: ", err)
	} else {
		logger.Info("Simulation successful. Putting HTML object up to s3")

		url, err := PutFile(destinationFile)

		if err == nil {
			s.ChannelMessageSend(m.ChannelID, "Done simulating. Check your report at "+url)
		} else {
			logger.Error("Received error when trying to put object to s3: ", err)
		}
	}
}

type tokenRefresh struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int32  `json:"expires_in"`
	Scope       string `json:"scope"`
}

type tokenRefreshError struct {
	ErrorType        string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func refreshAccessToken(accessToken tokenRefresh) {
	var err error
	for {
		if err != nil {
			log.Error("Received an error during the last token refresh", err)
			time.Sleep(time.Duration(10) * time.Second)
			err = nil
		} else {
			// Sleep for half of the expires time
			time.Sleep(time.Duration(accessToken.ExpiresIn/2) * time.Second)
		}

		log.Info("Refreshing battle-net token")

		accessToken, err = getNewAccessToken()
		if err != nil {
			err = setAccessTokenForSimcraft(accessToken)
		}
	}
}

func setAccessTokenForSimcraft(accessToken tokenRefresh) error {
	bnetAccessToken = accessToken.AccessToken

	usr, err := user.Current()
	if err != nil {
		return err
	}

	destination := path.Join(usr.HomeDir, ".simc-apitoken")

	d1 := []byte(accessToken.AccessToken)
	return ioutil.WriteFile(destination, d1, 0644)
}

func getNewAccessToken() (tokenRefresh, error) {
	tokenRefreshItem := tokenRefresh{}

	req, err := http.NewRequest("GET", "https://us.battle.net/oauth/token?grant_type=client_credentials", nil)

	req.Header.Add("Authorization", "Basic "+basicAuth(bnetClientID, bnetClientSecret))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)

	if err != nil {
		return tokenRefreshItem, err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		parsedError := &tokenRefreshError{}
		json.NewDecoder(resp.Body).Decode(parsedError)
		return tokenRefreshItem, errors.New(parsedError.ErrorType + ": " + parsedError.ErrorDescription)
	}

	if resp.StatusCode < 300 && resp.StatusCode >= 200 {
		json.NewDecoder(resp.Body).Decode(&tokenRefreshItem)
		log.Info("New access token fetched: " + tokenRefreshItem.AccessToken)
		if tokenRefreshItem.AccessToken == "" {
			return tokenRefreshItem, errors.New("No token set in response")
		}
		return tokenRefreshItem, nil
	}

	return tokenRefreshItem, errors.New("Unknown error occurred")
}

func basicAuth(username string, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
