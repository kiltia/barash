package meta

// Request query parameters for the Meta API endpoint.
type MetaRequest struct {
	Duns         string  `json:"duns"          ch:"duns"`
	Url          string  `json:"url"           ch:"url"`
	Name         *string `json:"name"          ch:"name"`
	LocAddress1  *string `json:"loc_address1"  ch:"loc_address1"`
	LocAddress2  *string `json:"loc_address2"  ch:"loc_address2"`
	MailAddress1 *string `json:"mail_address1" ch:"mail_address1"`
	MailAddress2 *string `json:"mail_address2" ch:"mail_address2"`
	MailCity     *string `json:"mail_city"     ch:"mail_city"`
	LocCity      *string `json:"loc_city"      ch:"loc_city"`
	LocState     *string `json:"loc_state"     ch:"loc_state"`
	MailState    *string `json:"mail_state"    ch:"mail_state"`
	MailZip      *string `json:"mail_zip"      ch:"mail_zip"`
	LocZip       *string `json:"loc_zip"       ch:"loc_zip"`
	MailCountry  *string `json:"mail_country"  ch:"mail_country"`
	LocCountry   *string `json:"loc_country"   ch:"loc_country"`
}
