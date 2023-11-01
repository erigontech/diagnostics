package api

/*func retrievePinFromURL(r *http.Request) (pins []uint64, err error) {

	for _, session := range strings.Split(r.URL.Query().Get("sessions"), ",") {
		pin, err := strconv.ParseUint(session, 10, 64)

		if err != nil {
			log.Printf("Error parsing session pin %s: %v\n", session, err)
			return pins, fmt.Errorf("error parsing session pin %s: %w", session, err)
		}

		pins = append(pins, pin)
	}

	if len(pins) == 0 {
		err = fmt.Errorf("no sessions")
	}

	return pins, err
}

func retrieveSizeStrFrom(r *http.Request) (uint64, error) {
	sizeStr := r.Form.Get("size")
	size, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing size %s: %w", sizeStr, err)
	}

	var offset uint64
	if size > 16*1024 {
		offset = size - 16*1024
	}

	return offset, nil
}*/
