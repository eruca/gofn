package main

import (
	"os"
)

const (
	T_parenthesis = 1
	T_brace       = 2
)

type Stack struct {
	i    int
	data []int
	bUse bool
}

func (s *Stack) push(n int) {
	if s.i+1 > len(s.data) {
		sint := make([]int, len(s.data))
		s.data = append(sint, s.data...)
		s.data = append(s.data, n)
	} else {
		s.data[s.i] = n
	}
	s.i++
}

func (s *Stack) pop() (n int) {
	n = s.data[s.i-1]
	s.data[s.i-1] = 0
	s.i--
	return
}

//util func

func isExist(file string) bool {
	_, err := os.Stat(file)
	return err == nil || os.IsExist(err)
}
