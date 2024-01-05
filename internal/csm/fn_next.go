package csm

func (csm *CronStateMachine) next() {
	if !csm.second.Next() {
		return
	}
	if !csm.minute.Next() {
		return
	}
	if !csm.hour.Next() {
		return
	}

	// Dates (dd-mm-yy) can be invalid in the case of leap years!
	// If an invalid date is detected, re-run the loop
	for next := true; next; next = !csm.isValidDate() {
		if !csm.day.Next() {
			continue
		}
		if !csm.month.Next() {
			if !csm.day.isValid() { // Can only happen with weekdays
				csm.day.Reset()
			}
			continue
		}
		if !csm.year.Next() {
			if !csm.day.isValid() { // Can only happen with weekdays
				csm.day.Reset()
			}
			continue
		}
	}
}

func (csm *CronStateMachine) isValidDate() bool {
	return csm.day.Value() <= csm.day.max()
}
