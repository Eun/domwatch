package api1

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
)

type Domain struct {
	ID          uint   `gorm:"primary_key;not null"`
	Domain      string `gorm:"type:char(255);unique;not null"`
	LastChecked int64  `gorm:"not null"`
	CreatedAt   time.Time
}

type Email struct {
	ID        uint   `gorm:"primary_key;not null"`
	Email     string `gorm:"type:char(255);unique;not null"`
	CreatedAt time.Time
}

type Watch struct {
	DomainID  uint `gorm:"not null"`
	EmailID   uint `gorm:"not null"`
	CreatedAt time.Time
}

func (api *API) watchRoute(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.Method, "POST") == false {
		api.writeError(w, "Must be a POST request")
		return
	}

	var err error
	apiRequest := struct {
		Domains []string
		Email   string
	}{}
	redirect := false
	contentType := r.Header.Get("Content-Type")
	if strings.EqualFold(contentType, "application/json") {
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&apiRequest)
		if err != nil {
			api.writeError(w, "invalid request")
			return
		}
	} else if strings.EqualFold(contentType, "application/x-www-form-urlencoded") {
		d := r.FormValue("domain")
		if len(d) <= 0 {
			if redirect {
				w.Header().Set("Location", "/#invalid_domain")
				w.WriteHeader(302)
			} else {
				api.writeError(w, "invalid domain")
			}
			return
		}
		apiRequest.Domains = []string{d}
		apiRequest.Email = r.FormValue("email")
		redirect = true
	} else {
		api.writeError(w, "invalid request")
		return
	}

	apiRequest.Email = strings.ToLower(apiRequest.Email)

	if !govalidator.IsEmail(apiRequest.Email) {
		if redirect {
			w.Header().Set("Location", "/#invalid_email")
			w.WriteHeader(302)
		} else {
			api.writeError(w, "invalid email")
		}
		return
	}

	var email Email
	err = api.db.FirstOrCreate(&email, &Email{Email: apiRequest.Email}).Error
	if err != nil {
		api.logError(w, err)
		return
	}

	for _, d := range apiRequest.Domains {
		if !govalidator.IsDNSName(d) {
			if redirect {
				w.Header().Set("Location", "/#invalid_domain")
				w.WriteHeader(302)
			} else {
				api.writeError(w, "invalid domain")
			}
			return
		}

		var domain Domain
		err = api.db.FirstOrCreate(&domain, &Domain{Domain: strings.ToLower(d)}).Error
		if err != nil {
			api.logError(w, err)
			return
		}

		var watch Watch
		err = api.db.FirstOrCreate(&watch, &Watch{DomainID: domain.ID, EmailID: email.ID}).Error
		if err != nil {
			api.logError(w, err)
			return
		}
	}

	if redirect {
		w.Header().Set("Location", "/#success")
		w.WriteHeader(302)
	} else {
		api.writeSuccessResponse(w, nil)
	}
}

func (api *API) unwatchRoute(w http.ResponseWriter, r *http.Request) {
	if strings.EqualFold(r.Method, "POST") == false {
		api.writeError(w, "Must be a POST request")
		return
	}

	var err error
	apiRequest := struct {
		Domains []string
		Email   string
	}{}
	redirect := false
	contentType := r.Header.Get("Content-Type")
	if strings.EqualFold(contentType, "application/json") {
		decoder := json.NewDecoder(r.Body)
		err = decoder.Decode(&apiRequest)
		if err != nil {
			api.writeError(w, "invalid request")
			return
		}
	} else if strings.EqualFold(contentType, "application/x-www-form-urlencoded") {
		d := r.FormValue("domain")
		if len(d) <= 0 {
			if redirect {
				w.Header().Set("Location", "/#invalid_domain")
				w.WriteHeader(302)
			} else {
				api.writeError(w, "invalid domain")
			}
			return
		}
		apiRequest.Domains = []string{d}
		apiRequest.Email = r.FormValue("email")
		redirect = true
	} else {
		api.writeError(w, "invalid request")
		return
	}

	apiRequest.Email = strings.ToLower(apiRequest.Email)

	if !govalidator.IsEmail(apiRequest.Email) {
		if redirect {
			w.Header().Set("Location", "/#invalid_email")
			w.WriteHeader(302)
		} else {
			api.writeError(w, "invalid email")
		}
		return
	}

	var email Email
	db := api.db.Where(&Email{Email: apiRequest.Email}).First(&email)
	if db.Error != nil {
		if db.RecordNotFound() {
			if redirect {
				w.Header().Set("Location", "/#success")
				w.WriteHeader(302)
			} else {
				api.writeSuccessResponse(w, nil)
			}
		} else {
			api.logError(w, err)
		}
		return
	}

	for _, d := range apiRequest.Domains {
		if !govalidator.IsDNSName(d) {
			if redirect {
				w.Header().Set("Location", "/#invalid_domain")
				w.WriteHeader(302)
			} else {
				api.writeError(w, "invalid domain")
			}
			return
		}
		var domain Domain
		db = api.db.Where(&Domain{Domain: strings.ToLower(d)}).First(&domain)
		if db.Error != nil {
			if db.RecordNotFound() {
				if redirect {
					w.Header().Set("Location", "/#success")
					w.WriteHeader(302)
				} else {
					api.writeSuccessResponse(w, nil)
				}
			} else {
				api.logError(w, err)
			}
			return
		}

		var watch Watch
		db = api.db.Where(&Watch{DomainID: domain.ID, EmailID: email.ID}).First(&watch)
		if db.Error != nil {
			if db.RecordNotFound() {
				if redirect {
					w.Header().Set("Location", "/#success")
					w.WriteHeader(302)
				} else {
					api.writeSuccessResponse(w, nil)
				}
			} else {
				api.logError(w, err)
			}
			return
		}

		err = api.db.Delete(&watch).Error
		if err != nil {
			api.logError(w, err)
			return
		}
	}
	if redirect {
		w.Header().Set("Location", "/#success")
		w.WriteHeader(302)
	} else {
		api.writeSuccessResponse(w, nil)
	}
}
