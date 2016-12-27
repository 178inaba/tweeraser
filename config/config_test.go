package config_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/178inaba/tweeraser/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	fileStr := `consumer_key = "foo"
consumer_secret = "bar"
access_token = "baz"
access_token_secret = "foobar"
`
	_, err = file.WriteString(fileStr)
	assert.NoError(t, err)
	file.Close()

	conf, err := config.LoadConfig(file.Name())
	assert.NoError(t, err)

	assert.Equal(t, "foo", conf.ConsumerKey)
	assert.Equal(t, "bar", conf.ConsumerSecret)
	assert.Equal(t, "baz", conf.AccessToken)
	assert.Equal(t, "foobar", conf.AccessTokenSecret)

	conf, err = config.LoadConfig("path/nothing.toml")
	assert.Nil(t, conf)
	assert.Error(t, err)
}
