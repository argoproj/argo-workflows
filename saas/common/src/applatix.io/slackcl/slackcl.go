package slackcl

import (
	"applatix.io/axerror"
	"github.com/nlopes/slack"
	"time"
)

const (
	EmailCacheRefreshIntervalInSecs = 120
)

type SlackClient struct {
	oauthToken       string
	client           *slack.Slack
	emailToNameCache map[string]string
	cacheLastRefresh int64
}

func New(oauthToken string) *SlackClient {
	client := slack.New(oauthToken)
	return &SlackClient{
		oauthToken:       oauthToken,
		client:           client,
		emailToNameCache: make(map[string]string),
		cacheLastRefresh: 0}
}

func (c *SlackClient) GetChannels() ([]string, *axerror.AXError) {
	channels, err := c.client.GetChannels(true)
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Slack API call failed with error: %v", err)
	}
	result := []string{}
	for _, channel := range channels {
		result = append(result, channel.Name)
	}
	return result, nil
}

func (c *SlackClient) GetUsers() (map[string]string, *axerror.AXError) {
	users, err := c.client.GetUsers()
	if err != nil {
		return nil, axerror.ERR_AX_INTERNAL.NewWithMessagef("Slack API call failed with error: %v", err)
	}
	result := make(map[string]string)
	for _, user := range users {
		result[user.Profile.Email] = user.Name
	}
	return result, nil
}

func (c *SlackClient) GetUserForEmail(email string, useCache bool) (string, *axerror.AXError) {

	// refresh the email-to-name cache if expired or if useCache flag is not set
	if !useCache || (c.cacheLastRefresh+EmailCacheRefreshIntervalInSecs <= time.Now().Unix()) {
		newCache, err := c.GetUsers()
		if err != nil {
			return "", err
		}
		c.emailToNameCache = newCache
		c.cacheLastRefresh = time.Now().Unix()
	}
	u, _ := c.emailToNameCache[email]
	return u, nil
}

func (c *SlackClient) PostMessageToChannel(channel string, text string) *axerror.AXError {
	return c.postMessage("#"+channel, text)
}

func (c *SlackClient) PostDirectMessage(username string, text string) *axerror.AXError {
	return c.postMessage("@"+username, text)
}

func (c *SlackClient) postMessage(channel string, text string) *axerror.AXError {
	_, _, err := c.client.PostMessage(channel, text, slack.NewPostMessageParameters())
	if err != nil {
		return axerror.ERR_AX_INTERNAL.NewWithMessagef("Slack API call failed with error: %v", err)
	}
	return nil
}
