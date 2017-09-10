package main

import (
	"github.com/willf/bloom"
)

const BLOOMSCORER_SIZE = 10000 // bits
const BLOOMSCORER_HASHES = 4   // no of hashes

type BloomScorer struct {
	filters []*bloom.BloomFilter
}

func NewBloomScorer(l int) BloomScorer {

	bs := BloomScorer{}
	bs.filters = make([]*bloom.BloomFilter, 0)

	for i := 0; i < l; i++ {
		bs.filters = append(bs.filters, bloom.New(BLOOMSCORER_SIZE, BLOOMSCORER_HASHES))
	}

	return bs
}

func (bs BloomScorer) Score(ve []string) float32 {

	if len(bs.filters) < len(ve) {
		// add moar filters
		for i := 0; i < len(ve)-len(bs.filters); i++ {
			bs.filters = append(bs.filters, bloom.New(BLOOMSCORER_SIZE, BLOOMSCORER_HASHES))
		}
	}

	s := float32(0)
	for i := 0; i < len(ve); i++ {
		if !bs.filters[i].TestAndAddString(ve[i]) {
			s++
		}
	}

	return s / float32(len(ve))

}
