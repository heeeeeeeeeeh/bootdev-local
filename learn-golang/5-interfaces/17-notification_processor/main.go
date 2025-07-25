package main

type notification interface {
	importance() int
}

type directMessage struct {
	senderUsername string
	messageContent string
	priorityLevel  int
	isUrgent       bool
}

type groupMessage struct {
	groupName      string
	messageContent string
	priorityLevel  int
}

type systemAlert struct {
	alertCode      string
	messageContent string
}

func (d directMessage) importance() int {
	importance := d.priorityLevel
	if d.isUrgent {
		importance = 50
	}
	return importance
}

func (g groupMessage) importance() int {
	return g.priorityLevel
}

func (s systemAlert) importance() int {
	return 100
}

func processNotification(n notification) (string, int) {
	switch a := n.(type) {
	case directMessage:
		return a.senderUsername, a.importance()
	case groupMessage:
		return a.groupName, a.importance()
	case systemAlert:
		return a.alertCode, a.importance()
	default:
		return "", 0
	}
}
