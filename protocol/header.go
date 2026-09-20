package protocol

type Header map[string][]string

func (h Header) Get(key string) string {
	values, ok := h[key]
	if !ok || len(values) == 0 {
		return ""
	} else {
		return values[0]
	}
}

// TODO
func (h Header) Set(key, value string) {
	_, ok := h[key]
	if !ok {
		h[key] = []string{value}
	} else {
		h[key] = append(h[key], value)
	}
}

// TODO
func (h Header) Add(key, value string) {

}

// TODO
func (h Header) Del(key string) {

}

// TODO
func (h Header) Values(key string) []string {
	return nil
}
