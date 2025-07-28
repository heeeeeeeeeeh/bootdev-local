package main

func countConnections(groupSize int) int {
	con := 0
	for i := range groupSize {
		for j := i + 1; j < groupSize; j++ {
			con++
		}
	}
	return con
}
