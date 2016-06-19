package instructor

import "strconv"

func stringToBool(s string) (interface{}, error) {
	return strconv.ParseBool(s)
}

func stringToPBool(s string) (interface{}, error) {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func stringToInt(s string) (interface{}, error) {
	return strconv.Atoi(s)
}

func stringToPInt(s string) (interface{}, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func stringToUint(s string) (interface{}, error) {
	return strconv.ParseUint(s, 10, 64)
}

func stringToPUint(s string) (interface{}, error) {
	i, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

func stringToFloat64(s string) (interface{}, error) {
	return strconv.ParseFloat(s, 64)
}

func stringToPFloat64(s string) (interface{}, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &f, err
}

func stringToString(s string) (interface{}, error) {
	return s, nil
}

func stringToPString(s string) (interface{}, error) {
	return &s, nil
}
