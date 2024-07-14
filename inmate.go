package main

import (
	"fmt"
	"log"
)

type Case struct {
	CaseNo     string  `json:"caseNo"`
	Status     string  `json:"status"`
	BondType   string  `json:"bondType"`
	BondAmount float64 `json:"bondAmount"`
	FineAmount float64 `json:"fineAmount"`
	Sentence   string  `json:"sentence"`  // "0y 0m 0d"
	CourtTime  string  `json:"courtTime"` // "Jul 10 2024 9:00AM"
}

// Currently lack type data for certain fields
type Charge struct {
	// "case": null,
	Case string `json:"case"`
	// "caseNo": null,
	CaseNo string `json:"caseNo"`
	// "crimeType": null,
	CrimeType string `json:"crimeType"`
	// "counts": null,
	// "modifier": null,
	// "controlNumber": null,
	ControlNumber string `json:"controlNumber"`
	// "warrantNumber": null,
	WarrantNumber string `json:"warrantNumber"`
	// "arrestCode": null,
	ArrestCode string `json:"arrestCode"`
	// "chargeDescription": "BURGLARY: COMMERCIAL BUILDINGS,CARS,ETC.",
	ChargeDescription string `json:"chargeDescription"`
	// "bondType": "NB",
	BondType string `json:"bondType"`
	// "bondAmount": "0.00",
	BondAmount string `json:"bondAmount"`
	// "courtType": "Circuit Biloxi ",
	CourtType string `json:"courtType"`
	// "courtTime": null,
	CourtTime string `json:"courtTime"`
	// "courtName": "",
	CourtName string `json:"courtName"`
	// "chargeStatus": "AWAITING COURT",
	ChargeStatus string `json:"chargeStatus"`
	// "offenseDate": "2024-06-28",
	OffenseDate string `json:"offenseDate"`
	// "arrestDate": "2024-06-28",
	ArrestDate string `json:"arrestDate"`
	// "arrestingAgency": "Circuit Court"
	ArrestingAgency string `json:"arrestingAgency"`
}

// Currently lack data
// type Hold struct{}
type Hold map[string]interface{}

// We care about:
// "Sched Release" (date?)
// "Booking Date" (datetime "6/28/2024 10:22:44 AM")
// "Date Released" (date?)
// "Arrest Date" (date "6/28/2024")
// "Arresting Agency" (string "Circuit Court")
type SpecialField struct {
	LabelText string `json:"labelText"`
	Value     string `json:"offenderValue"`
}

// This is the per-inmate response format, which differs from the overall jail response's "offenders" list.
type InmateResponse struct {
	// Related only to the request, not the inmate.
	CaptchaRequired bool   `json:"captchaRequred"` // sic, yes "Requred"
	OffenderViewKey int    `json:"offenderViewKey"`
	CaptchaKey      string `json:"captchaKey"`
	ErrorMessage    string `json:"errorMessage"`
	Success         bool   `json:"succes"` // sic, yes "succes"
	// Related to the actual inmate
	Cases         []Case         `json:"cases"`
	Charges       []Charge       `json:"charges"`
	Holds         []Hold         `json:"holds"`
	SpecialFields []SpecialField `json:"offenderSpecialFields"`
}

type Inmate struct {
	ArrestNo             string `json:"arrestNo"`
	OriginalBookDateTime string `json:"originalBookDateTime"` // "6/28/2024T10:22:44"
	FinalReleaseDateTime string `json:"finalReleaseDateTime"` // Presumably the same format?
	AgencyName           string `json:"agencyName"`
	Jacket               string `json:"jacket"`

	// In initial testing, these are null on initial parse from the "offenders" list
	Cases   []Case   `json:"cases"`
	Charges []Charge `json:"charges"`
	Holds   []Hold   `json:"holds"`

	// We ignore offenderSpecialFields here (null on initial parse from the "offenders" list)
	// and instead parse these from the per-inmate response.
	// "Sched Release" (date?)
	SpecialSchedRelease string `json:"specialSchedRelease"`
	// "Booking Date" (datetime "6/28/2024 10:22:44 AM"; differs from OriginalBookDateTime)
	// Note that this is probably set whenever OriginalBookDateTime is set,
	// so if this isn't set, we probably failed to get the individual inmate info.
	// (In which case we may incorrectly see 0 charges.)
	SpecialBookingDate string `json:"specialBookingDate"`
	// "Date Released" (date?)
	SpecialDateReleased string `json:"specialDateReleased"`
	// "Arrest Date" (date "6/28/2024")
	SpecialArrestDate string `json:"specialArrestDate"`
	// "Arresting Agency" (string "Circuit Court")
	SpecialArrestingAgency string `json:"specialArrestingAgency"`
	// "Arresting Officer" (string "SOME NAME")
	SpecialArrestingOfficer string `json:"specialArrestingOfficer"`
}

func (i *Inmate) Update(j *Jail) error {
	//"<OMS_URL>/jtclientweb/Offender/<JAIL_NAME>/<ARREST_NO>/offenderbucket/<OFFENDER_VIEW_KEY>",
	inmateURL := fmt.Sprintf("%s/jtclientweb/Offender/%s/%s/offenderbucket/%d",
		j.BaseURL,
		j.Name,
		i.ArrestNo,
		j.OffenderViewKey,
	)
	payload := &CaptchaProtocol{
		CaptchaKey:   j.CaptchaKey,
		CaptchaImage: "",
		UserCode:     "",
	}
	inmateResponse := &InmateResponse{}

	// We can only make so many requests for data before we need to solve a captcha again.
	// Here, we try to solve the captcha and then retry the request once.
	// (Note: this seems to not always be the case, but it's not clear to me what triggers it.
	// Sometimes I immediately get captcha'd every 5 requests, sometimes it's only on the first one.)
	for attempt := 0; attempt < 2; attempt++ {
		err := PostJSON[CaptchaProtocol, InmateResponse](inmateURL, nil, payload, inmateResponse)
		if err != nil {
			return fmt.Errorf("failed to update inmate: %w", err)
		}
		if !inmateResponse.CaptchaRequired { // Success!
			break
		}
		if attempt == 0 { // Try to refresh captcha
			log.Printf("Captcha required for inmate \"%s\"; refreshing", i.ArrestNo)
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
	i.Cases = inmateResponse.Cases
	i.Charges = inmateResponse.Charges
	i.Holds = inmateResponse.Holds
	for _, specialField := range inmateResponse.SpecialFields {
		switch specialField.LabelText {
		case "Sched Release":
			i.SpecialSchedRelease = specialField.Value
		case "Booking Date":
			i.SpecialBookingDate = specialField.Value
		case "Date Released":
			i.SpecialDateReleased = specialField.Value
		case "Arrest Date":
			i.SpecialArrestDate = specialField.Value
		case "Arresting Agency":
			i.SpecialArrestingAgency = specialField.Value
		case "Arresting Officer":
			i.SpecialArrestingOfficer = specialField.Value
		}
	}

	return nil
}
