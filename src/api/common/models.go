package common

type JsonString string

func (js *JsonString) UnmarshalJSON(b []byte) error {
	*js = JsonString(b)
	return nil
}
