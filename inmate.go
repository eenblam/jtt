package main

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
