package api

type JSONString string

func (js *JSONString) UnmarshalJSON(b []byte) error {
	*js = JSONString(b)
	return nil
}

