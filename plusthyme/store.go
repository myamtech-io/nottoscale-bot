package plusthyme

import ()

var (
	store map[string]bool
)

func init() {
	store = make(map[string]bool)
}

func GetAllRegistered() []string {
	ret := make([]string, len(store))

	var i = 0
	for key, _ := range store {
		ret[i] = key
		i++
	}

	return ret
}

func UpdateRegistration(userID string, isRegistered bool) {
	if !isRegistered {
		delete(store, userID)
	} else {
		store[userID] = isRegistered
	}
}
