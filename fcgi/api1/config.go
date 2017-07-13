package api1

import (
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	hjson "github.com/hjson/hjson-go"
	"github.com/mitchellh/mapstructure"
)

type MailConfig struct {
	Sender   *string
	Server   *string
	Port     *int
	Username *string
	Password *string
	Auth     *string
}

type Config struct {
	Mail             MailConfig
	mailAuth         smtp.Auth
	CheckInterval    *string
	intervalDuration time.Duration
	DNSServer        *string
}

// NewConfigFromMap creates a new config instance from a map
func NewConfigFromMap(dat map[string]interface{}) (*Config, error) {
	var config Config
	err := mapstructure.Decode(dat, &config)
	if err != nil {
		return nil, err
	}
	err = config.SetDefaults()
	return &config, err
}

// NewConfigFromFile returns a Config instance for a file
func NewConfigFromFile(configFile string) (*Config, error) {
	bytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var dat map[string]interface{}
	if err := hjson.Unmarshal(bytes, &dat); err != nil {
		return nil, err
	}

	return NewConfigFromMap(dat)
}

func (config *Config) SetDefaults() (err error) {
	if config.Mail.Server == nil {
		return errors.New("No Mail server defined")
	}
	if config.Mail.Sender == nil {
		return errors.New("No Mail sender defined")
	}
	if config.Mail.Password == nil {
		config.Mail.Password = new(string)
	}
	if config.Mail.Username == nil {
		config.Mail.Username = new(string)
	}
	if config.Mail.Port == nil {
		config.Mail.Port = new(int)
		*config.Mail.Port = 25
	}

	if config.DNSServer == nil {
		config.DNSServer = new(string)
		*config.DNSServer = "8.8.8.8"
	}

	if config.Mail.Auth != nil && strings.ToLower(*config.Mail.Auth) == "cram-md5" {
		config.mailAuth = smtp.CRAMMD5Auth(*config.Mail.Username, *config.Mail.Password)
	} else {
		config.mailAuth = smtp.PlainAuth("", *config.Mail.Username, *config.Mail.Password, *config.Mail.Server)
	}

	// test auth
	var client *smtp.Client
	client, err = smtp.Dial(*config.Mail.Server + ":" + strconv.Itoa(*config.Mail.Port))
	if err != nil {
		return err
	}
	defer client.Close()
	client.Hello("localhost")

	if ok, _ := client.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: *config.Mail.Server}
		if err = client.StartTLS(config); err != nil {
			return err
		}
	}
	if ok, _ := client.Extension("AUTH"); ok {
		if err = client.Auth(config.mailAuth); err != nil {
			return err
		}
	}

	if config.CheckInterval == nil {
		config.intervalDuration, _ = time.ParseDuration("6h")
	} else {
		config.intervalDuration, err = time.ParseDuration(*config.CheckInterval)
	}
	return err
}
