package stacks

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type Stack struct {
	Number   int
	State    string
	WaitTime time.Duration
	Frames   []Frame
}

func (s *Stack) Print() {
	state := s.State
	if s.WaitTime != 0 {
		state += ", " + s.WaitTime.String()
	}
	fmt.Printf("goroutine %d [%s]:\n", s.Number, s.WaitTime)
	for _, f := range s.Frames {
		f.Print()
	}

	fmt.Println()
}

type Frame struct {
	Function string
	Params   []string
	File     string
	Line     int
}

func (f *Frame) Print() {
	fmt.Println(f.Function, f.Params)
	fmt.Printf("\t%s:%d\n", f.File, f.Line)
}

type Filter func(s *Stack) bool

func HasFrameMatching(pattern string) Filter {
	return func(s *Stack) bool {
		for _, f := range s.Frames {
			if strings.Contains(f.Function, pattern) || strings.Contains(f.File, pattern) {
				return true
			}
		}
		return false
	}
}

func TimeGreaterThan(d time.Duration) Filter {
	return func(s *Stack) bool {
		return s.WaitTime >= d
	}
}

func Negate(f Filter) Filter {
	return func(s *Stack) bool {
		return !f(s)
	}
}

func ApplyFilters(stacks []*Stack, filters []Filter) []*Stack {
	var out []*Stack

next:
	for _, s := range stacks {
		for _, f := range filters {
			if !f(s) {
				continue next
			}
		}
		out = append(out, s)
	}
	return out
}

func ParseStacks(r io.Reader) ([]*Stack, error) {

	var cur *Stack
	var stacks []*Stack
	var frame *Frame
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		if strings.HasPrefix(scan.Text(), "goroutine") {
			parts := strings.Split(scan.Text(), " ")
			num, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("unexpected formatting: %s", scan.Text())
			}

			var timev time.Duration
			state := strings.Split(strings.Trim(strings.Join(parts[2:], " "), "[]:"), ",")
			if len(state) > 1 {
				timeparts := strings.Fields(state[1])
				if len(timeparts) != 2 {
					return nil, fmt.Errorf("weirdly formatted time string: %q", state[1])
				}

				val, err := strconv.Atoi(timeparts[0])
				if err != nil {
					return nil, err
				}

				timev = time.Duration(val) * time.Minute
			}

			cur = &Stack{
				Number:   num,
				State:    state[0],
				WaitTime: timev,
			}
			continue
		}
		if scan.Text() == "" {
			stacks = append(stacks, cur)
			cur = nil
			continue
		}

		if frame == nil {
			frame = &Frame{
				Function: scan.Text(),
			}

			n := strings.LastIndexByte(scan.Text(), '(')
			if n > -1 {
				frame.Function = scan.Text()[:n]
				frame.Params = strings.Fields(scan.Text()[n+1 : len(scan.Text())-1])
			}

		} else {
			parts := strings.Split(scan.Text(), ":")
			frame.File = strings.Trim(parts[0], " \t\n")
			if len(parts) != 2 {
				fmt.Printf("expected a colon: %q\n", scan.Text())
				os.Exit(1)
			}

			lnum, err := strconv.Atoi(strings.Split(parts[1], " ")[0])
			if err != nil {
				return nil, fmt.Errorf("error finding line number: ", scan.Text())
			}

			frame.Line = lnum
			cur.Frames = append(cur.Frames, *frame)
			frame = nil
		}
	}

	if cur != nil {
		stacks = append(stacks, cur)
	}

	return stacks, nil
}

type StackCompFunc func(a, b *Stack) bool
type StackSorter struct {
	Stacks   []*Stack
	CompFunc StackCompFunc
}

func (ss StackSorter) Len() int {
	return len(ss.Stacks)
}

func (ss StackSorter) Less(i, j int) bool {
	return ss.CompFunc(ss.Stacks[i], ss.Stacks[j])
}

func (ss StackSorter) Swap(i, j int) {
	ss.Stacks[i], ss.Stacks[j] = ss.Stacks[j], ss.Stacks[i]
}

func CompWaitTime(a, b *Stack) bool {
	return a.WaitTime < b.WaitTime
}

func CompDepth(a, b *Stack) bool {
	return len(a.Frames) < len(b.Frames)
}

func CompGoroNum(a, b *Stack) bool {
	return a.Number < b.Number
}
