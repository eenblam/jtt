package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

const DefaultDomainName = "omsweb.public-safety-cloud.com"

type JailResponse struct {
	// Yes, "Requred", this is in their API.
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
	for i := 0; i < MAX_CAPTCHA_ATTEMPTS; i++ {
		captchaKey, err = ProcessCaptcha(j)
		if err != nil {
			log.Printf("failed to solve captcha: %v", err)
			continue
		}
		captchaMatched = true
		break
	}
	if !captchaMatched {
		return fmt.Errorf("failed to match captcha after %d attempts", MAX_CAPTCHA_ATTEMPTS)
	}
	j.CaptchaKey = captchaKey
	log.Println("Captcha matched!")

	return nil
}

func (j *Jail) UpdateInmate(inmate *Inmate) error {
	//"<OMS_URL>/jtclientweb/Offender/<JAIL_NAME>/<ARREST_NO>/offenderbucket/<OFFENDER_VIEW_KEY>",
	location := fmt.Sprintf("https://%s/jtclientweb/Offender/%s/%s/offenderbucket/%d",
		j.DomainName,
		j.Name,
		inmate.ArrestNo,
		j.OffenderViewKey,
	)
	payload := &CaptchaProtocol{
		CaptchaKey:   j.CaptchaKey,
		CaptchaImage: "",
		UserCode:     "",
	}
	inmateResponse := &InmateResponse{}
	for i := 0; i < 2; i++ { // Allow a retry with a fresh captcha
		err := RequestJSONIntoStruct[CaptchaProtocol, InmateResponse]("POST", location, nil, inmateResponse, payload)
		if err != nil {
			return fmt.Errorf("failed to update inmate: %w", err)

		}
		if !inmateResponse.CaptchaRequired { // Success!
			break
		}
		if i == 0 { // Try to refresh captcha
			err = j.updateCaptcha()
			if err != nil {
				return fmt.Errorf("failed to update inmate due to failed captcha: %w", err)
			}
		} else { // Already retried
			return fmt.Errorf("captcha required for inmate after refresh. Response: %v", inmateResponse)
		}
	}
	// Update the jail's view key
	j.OffenderViewKey = inmateResponse.OffenderViewKey
	// Update the inmate's data
	inmate.Cases = inmateResponse.Cases
	inmate.Charges = inmateResponse.Charges
	inmate.Holds = inmateResponse.Holds
	for _, specialField := range inmateResponse.SpecialFields {
		switch specialField.LabelText {
		case "Sched Release":
			inmate.SpecialSchedRelease = specialField.Value
		case "Booking Date":
			inmate.SpecialBookingDate = specialField.Value
		case "Date Released":
			inmate.SpecialDateReleased = specialField.Value
		case "Arrest Date":
			inmate.SpecialArrestDate = specialField.Value
		case "Arresting Agency":
			inmate.SpecialArrestingAgency = specialField.Value
		case "Arresting Officer":
			inmate.SpecialArrestingOfficer = specialField.Value
		}
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

	for i := range j.Offenders {
		// Chill out for a bit to be especially gentle to their server
		// Convert time.Second (duration in nanoseconds) to float, scale to 0.5-1.5 seconds
		duration := time.Duration((0.5 + rand.Float64()) * float64(time.Second))
		time.Sleep(duration)

		inmate := &j.Offenders[i]
		err := j.UpdateInmate(inmate)
		if err != nil {
			log.Printf("failed to update inmate \"%s\": %v", inmate.ArrestNo, err)
			continue
		}
		log.Printf("Updated inmate \"%s\". Cases: %d Charges: %d Holds: %d Booked: %s",
			inmate.ArrestNo, len(inmate.Cases), len(inmate.Charges), len(inmate.Holds), inmate.SpecialBookingDate,
		)
	}
	return j, nil
}
