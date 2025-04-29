package meta

import "time"

// Request query parameters for the Meta API endpoint.
type VerifyParams struct {
	Duns         string    `query:"duns"          ch:"duns"`
	Url          string    `query:"url"           ch:"url"`
	Dba          *string   `query:"dba"           ch:"dba"`
	Name         *string   `query:"name"          ch:"name"`
	LocAddress1  *string   `query:"loc_address1"  ch:"loc_address1"`
	LocAddress2  *string   `query:"loc_address2"  ch:"loc_address2"`
	MailAddress1 *string   `query:"mail_address1" ch:"mail_address1"`
	MailAddress2 *string   `query:"mail_address2" ch:"mail_address2"`
	MailCity     *string   `query:"mail_city"     ch:"mail_city"`
	LocCity      *string   `query:"loc_city"      ch:"loc_city"`
	LocState     *string   `query:"loc_state"     ch:"loc_state"`
	MailState    *string   `query:"mail_state"    ch:"mail_state"`
	MailZip      *string   `query:"mail_zip"      ch:"mail_zip"`
	LocZip       *string   `query:"loc_zip"       ch:"loc_zip"`
	MailCountry  *string   `query:"mail_country"  ch:"mail_country"`
	LocCountry   *string   `query:"loc_country"   ch:"loc_country"`
	Timestamp    time.Time `                      ch:"ts"`
}
