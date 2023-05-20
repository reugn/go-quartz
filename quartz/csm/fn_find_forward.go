package csm

func (csm *CronStateMachine) findForward() {
	// Initial find, checking from most to least significant
	nodes := []NodeID{years, months, days, hours, minutes, seconds}
	for _, nodeID := range nodes {
		node := csm.selectNode(nodeID)
		if ffresult := node.findForward(); ffresult != unchanged {
			csm.resetFrom(nodeID - 1)
			if ffresult == overflowed {
				csm.overflowFrom(nodeID + 1)
			}
			return
		}
	}

	// If no changes were applied, advance from least to most significant
	csm.next()
}

// Reset all nodes below and including this one
func (csm *CronStateMachine) resetFrom(node NodeID) {
	chosenNode := csm.selectNode(node)
	if chosenNode == nil {
		return
	}

	chosenNode.Reset()
	csm.resetFrom(node - 1)
}

// Advance all nodes above and including this one
func (csm *CronStateMachine) overflowFrom(node NodeID) {
	chosenNode := csm.selectNode(node)
	if chosenNode == nil {
		return
	}

	if chosenNode.Next() { // if overflows, keep recursing
		csm.overflowFrom(node + 1) // Overflow above
	} else {
		csm.resetFrom(node - 1) // Reset below
	}
}

// Select node from enum
func (csm *CronStateMachine) selectNode(node NodeID) csmNode {
	switch node {
	case years:
		return csm.year
	case months:
		return csm.month
	case days:
		return csm.day
	case hours:
		return csm.hour
	case minutes:
		return csm.minute
	case seconds:
		return csm.second
	}
	return nil
}
