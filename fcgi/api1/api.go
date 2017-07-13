package api1

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"time"

	"encoding/json"

	"strconv"

	"github.com/Eun/domwatch"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	"github.com/miekg/dns"
)

type API struct {
	db        *gorm.DB
	closeChan chan bool
	config    *Config
	logger    *log.Logger
}

func NewApi(config *Config, db *gorm.DB, router *mux.Router, logger *log.Logger) (*API, error) {
	var api API
	api.db = db

	err := config.SetDefaults()
	if err != nil {
		return nil, err
	}

	api.config = config

	db.AutoMigrate(&Domain{})
	db.AutoMigrate(&Email{})
	db.AutoMigrate(&Watch{})

	router.HandleFunc("/stats", api.statsRoute)
	router.HandleFunc("/watch", api.watchRoute)
	router.HandleFunc("/unwatch", api.unwatchRoute)

	api.logger = logger

	return &api, nil
}

func (api *API) Run() error {
	api.closeChan = make(chan bool)
	go api.watchDomainsTask()
	return nil
}

func (api *API) Close() {
	api.closeChan <- true
}

func (api *API) writeError(w http.ResponseWriter, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	json.NewEncoder(w).Encode(&struct{ Error string }{err})
}

func (api *API) writeNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(404)
	json.NewEncoder(w).Encode(&struct{ Error string }{"not found"})
}

func (api *API) writeAccessDenied(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(403)
	json.NewEncoder(w).Encode(&struct{ Error string }{"access denied"})
}

func (api *API) writeSuccessResponse(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

func (api *API) logError(w http.ResponseWriter, err error) {
	id, _ := genUUID()
	api.logger.Printf("Error ID=%s: %s", id, err.Error())
	api.writeError(w, fmt.Sprintf("A wild error appeard, thankfully it was logged, its id is %s", id))
}

func (api *API) watchDomainsTask() {
	api.watchDomains()
	timer := time.NewTimer(api.config.intervalDuration)
	for {
		select {
		case <-api.closeChan:
			return
		case <-timer.C:
			api.watchDomains()
			timer = time.NewTimer(api.config.intervalDuration)
		}
	}
}

type devNullWriter struct {
}

func (*devNullWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (api *API) watchDomains() {
	api.logger.Println("Checking domains")
	now := time.Now().UTC().Unix()
	var domains []Domain
	err := api.db.Find(&domains).Error
	if err != nil {
		api.logger.Printf("Error on watchDomains: %s", err.Error())
		return
	}

	db := api.db

	var available bool
	for _, dom := range domains {

		// are there any watchers for this domain?
		var watches []Watch
		err := api.db.Where(&Watch{DomainID: dom.ID}).Find(&watches).Error
		if err != nil {
			continue
		}

		// if not delete it right away
		if len(watches) == 0 {
			db = db.Delete(&dom)
			continue
		}

		api.logger.Printf("Checking '%s'\n", dom.Domain)
		available, err = domwatch.IsDomainAvailable(*api.config.DNSServer, dom.Domain, "tcp", []uint16{dns.TypeNS, dns.TypeSOA}, api.logger)
		if err == nil && available {
			api.logger.Printf("'%s' is available\n", dom.Domain)
			err = api.notifyWatchers(watches, &dom)
			if err != nil {
				api.logger.Printf("Error on notifyUsers: %s", err.Error())
				return
			}
			db = db.Delete(&dom)
		} else {
			dom.LastChecked = now
			db = db.Save(&dom)
			if err != nil {
				api.logger.Printf("Error  for '%s': %s\n", dom.Domain, err.Error())
			}
		}

	}

	err = db.Error
	if err != nil {
		api.logger.Printf("Error on watchDomains: %s", err.Error())
		return
	}
}

func (api *API) notifyWatchers(watches []Watch, domain *Domain) error {

	for _, w := range watches {
		var email Email
		err := api.db.Where(&Email{ID: w.EmailID}).Find(&email).Error
		if err != nil {
			continue
		}
		err = api.notifyUser(&email, domain)
		if err != nil {
			api.logger.Printf("Error on notifyUsers: %s", err.Error())
		}
	}

	return nil
}

type smtpTemplateData struct {
	From         string
	To           string
	Domain       string
	Time         string
	OtherDomains []string
}

const emailTemplate = `From: {{.From}}
To: {{.To}}
Subject: ⚠️ {{.Domain}} is available!
Date: {{.Time}}

We just wanted to notify you that the domain

    {{.Domain}}

is now available.
{{if .OtherDomains}}
You are also subscribed to following domains: {{range $index, $element := .OtherDomains}}{{if $index}}, {{end}}{{$element}}{{end}}
{{end}}
Sincerely,

domwatch
`

func (api *API) notifyUser(email *Email, domain *Domain) (err error) {
	var doc bytes.Buffer
	context := &smtpTemplateData{
		*api.config.Mail.Sender,
		email.Email,
		domain.Domain,
		time.Now().UTC().Format(time.RFC1123Z),
		[]string{},
	}

	var watches []Watch
	err = api.db.Where(&Watch{EmailID: email.ID}).Find(&watches).Error
	if err == nil {
		for _, w := range watches {
			var d Domain
			err = api.db.Where(&Domain{ID: w.DomainID}).Find(&d).Error
			if err == nil && d.Domain != domain.Domain {
				context.OtherDomains = append(context.OtherDomains, d.Domain)
			}
		}
	}

	t := template.New("emailTemplate")
	t, err = t.Parse(emailTemplate)
	if err != nil {
		return err
	}
	err = t.Execute(&doc, context)
	if err != nil {
		return err
	}

	return smtp.SendMail(*api.config.Mail.Server+":"+strconv.Itoa(*api.config.Mail.Port),
		api.config.mailAuth,
		*api.config.Mail.Sender,
		[]string{email.Email},
		doc.Bytes())
}
