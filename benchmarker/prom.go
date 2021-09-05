package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/isucon/isucon11-final/benchmarker/scenario"
)

const promTagPrefix = "isucon11f_bench_"

type PromTags struct {
	b bytes.Buffer
}

func (p *PromTags) writeTag(name string, value string, labelsAndValues ...string) {
	if len(labelsAndValues)%2 != 0 {
		panic("len(labelsAndValues) needs to be a multiple of 2")
	}

	var labels []string
	for i := 0; i < len(labelsAndValues)/2; i++ {
		labels = append(labels, fmt.Sprintf("%s=\"%s\"", labelsAndValues[i*2], labelsAndValues[i*2+1]))
	}
	_, _ = p.b.WriteString(fmt.Sprintf("%s%s{%s} %s\n", promTagPrefix, name, strings.Join(labels, ","), value))
}

func (p *PromTags) writePromFile() {
	if len(promOut) == 0 {
		return
	}

	promOutNew := fmt.Sprintf("%s.new", promOut)
	err := os.WriteFile(promOutNew, p.b.Bytes(), 0644)
	if err != nil {
		scenario.AdminLogger.Printf("Failed to write prom file: %s", err)
		return
	}
}

func (p *PromTags) commit() {
	if len(promOut) == 0 {
		return
	}

	promOutNew := fmt.Sprintf("%s.new", promOut)
	err := os.Rename(promOutNew, promOut)
	if err != nil {
		scenario.AdminLogger.Printf("Failed to write prom file: %s", err)
		return
	}
}
