package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

const DefaultDomainName = "omsweb.public-safety-cloud.com"

type JailResponse struct {
	// Yes, "Requred", this is in their API. This key (like others) is typo'd.
	CaptchaRequired bool `json:"captchaRequred"`
	// They'll keep updating this
	CaptchaKey string `json:"captchaKey"`
	// The initial list of inmate data
	Offenders []Inmate `json:"offenders"`
	// This is updated with every request
	OffenderViewKey int `json:"offenderViewKey"`
}

type Jail struct {
	// Domain for the jail. Usually "omsweb.public-safety-cloud.com", but not always!
	DomainName string
	// Name of the jail, as it appears in the URL
	Name string
	// This is sent with each request, and sometimes updated
	CaptchaKey string
	//TODO rename "offenders" to something more appropriate; this is JailTracker terminology
	Offenders []Inmate
	// Each request (after validation) updates this key!
	OffenderViewKey int
}

func NewJail(domainName, name string) (*Jail, error) {
	j := &Jail{
		DomainName: domainName,
		Name:       name,
	}
	if err := j.updateCaptcha(); err != nil {
		return nil, fmt.Errorf("failed to update captcha: %w", err)
	}
	log.Println("Captcha matched!")

	// Make initial request for jail data
	payload := &CaptchaProtocol{
		CaptchaKey:   j.CaptchaKey,
		CaptchaImage: "",
		// This is normally null in this request in the web client :\
		UserCode: "",
	}
	jailResponse := &JailResponse{}
	err := RequestJSONIntoStruct("POST", j.getJailAPIURL(), nil, jailResponse, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to request initial jail data: %w", err)
	}
	if jailResponse.CaptchaRequired {
		return nil, fmt.Errorf("captcha required for jail. Response: %v", jailResponse)
	}

	j.OffenderViewKey = jailResponse.OffenderViewKey
	j.Offenders = jailResponse.Offenders
	return j, nil
}

func (j *Jail) updateCaptcha() error {
	captchaMatched := false
	var captchaKey string
	var err error
	for i := 0; i < MaxCaptchaAttempts; i++ {
		captchaKey, err = ProcessCaptcha(j)
		if err != nil {
			log.Printf("failed to solve captcha: %v", err)
			continue
		}
		captchaMatched = true
		break
	}
	if !captchaMatched {
		return fmt.Errorf("failed to match captcha after %d attempts", MaxCaptchaAttempts)
	}
	j.CaptchaKey = captchaKey
	log.Println("Captcha matched!")

	return nil
}

// UpdateInmates updates all inmates in the jail.
// Currently returns only a nil error, but reserving one here for future use.
func (j *Jail) UpdateInmates() error {
	for i := range j.Offenders {
		// Chill out for a bit to be especially gentle to their server
		// Convert time.Second (duration in nanoseconds) to float, scale to 0.5-1.5 seconds
		duration := time.Duration((0.5 + rand.Float64()) * float64(time.Second))
		time.Sleep(duration)

		inmate := &j.Offenders[i]
		//err := j.UpdateInmate(inmate)
		err := inmate.Update(j)
		if err != nil {
			log.Printf("failed to update inmate \"%s\": %v", inmate.ArrestNo, err)
			continue
		}
		log.Printf("Updated inmate \"%s\". Cases: %d Charges: %d Holds: %d Booked: %s",
			inmate.ArrestNo, len(inmate.Cases), len(inmate.Charges), len(inmate.Holds), inmate.SpecialBookingDate,
		)
	}
	return nil
}

// Get the URL for the jail's main page, as it would be accessed by a web browser.
// Jails have their own URL within the domain, but the captcha service needs to know which jail
// the captcha corresponds to, so it looks for this URL in the Referer header.
func (j Jail) getJailURL() string {
	return fmt.Sprintf("https://%s/jtclientweb/jailtracker/index/%s", j.DomainName, j.Name)
}

// Get the URL for the jail's JSON API, which will list all inmates.
func (j Jail) getJailAPIURL() string {
	return fmt.Sprintf("https://%s/jtclientweb/Offender/%s", j.DomainName, j.Name)
}

func CrawlJail(baseURL, name string) (*Jail, error) {
	log.Printf("Crawling jail: %s", name)
	j, err := NewJail(baseURL, name)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize jail: %w", err)
	}
	log.Printf("Found %d inmates", len(j.Offenders))

	err = j.UpdateInmates()
	if err != nil {
		return nil, fmt.Errorf("failed to update inmates: %w", err)
	}
	return j, nil
}
