package llmmeta

import (
	"encoding/json"

	"orb/runner/pkg/log"
)

type LlmTaskParams struct {
	Id              uint64  `ch:"id"`
	Duns            string  `ch:"duns"`
	Name            string  `ch:"name"`
	Url             string  `ch:"url"              query:"url"`
	Dba             *string `ch:"dba"`
	LocAddress1     *string `ch:"loc_address1"`
	LocAddress2     *string `ch:"loc_address2"`
	MailAddress1    *string `ch:"mail_address1"`
	MailAddress2    *string `ch:"mail_address2"`
	MailCity        *string `ch:"mail_city"`
	LocCity         *string `ch:"loc_city"`
	LocState        *string `ch:"loc_state"`
	MailState       *string `ch:"mail_state"`
	MailZip         *string `ch:"mail_zip"`
	LocZip          *string `ch:"loc_zip"`
	MailCountry     *string `ch:"mail_country"`
	LocCountry      *string `ch:"loc_country"`
	DiscoveryPrompt string  `ch:"discovery_prompt"`
	TaskPrompt      string  `ch:"task_prompt"`
	JsonSchema      string  `ch:"json_schema"`
}

func (p LlmTaskParams) GetBody() map[string]any {
	inputDataRepr, err := json.MarshalIndent(
		map[string]any{
			"name":          p.Name,
			"dba":           p.Dba,
			"loc_address1":  p.LocAddress1,
			"loc_address2":  p.LocAddress2,
			"loc_city":      p.LocCity,
			"loc_state":     p.LocState,
			"loc_country":   p.LocCountry,
			"loc_zip":       p.LocZip,
			"mail_address1": p.MailAddress1,
			"mail_address2": p.MailAddress2,
			"mail_city":     p.MailCity,
			"mail_state":    p.MailState,
			"mail_country":  p.MailCountry,
			"mail_zip":      p.MailZip,
		},
		"",
		"\t",
	)
	if err != nil {
		log.S.Error(
			"Failed to marshal task data into the JSON representation",
			log.L().Tag(log.LogTagDataProvider).Error(err).Add("data", p),
		)
	}

	return map[string]any{
		"discovery_prompt": p.DiscoveryPrompt,
		"task_prompt":      p.TaskPrompt,
		"json_schema":      p.JsonSchema,
		"task_data":        string(inputDataRepr),
	}
}
