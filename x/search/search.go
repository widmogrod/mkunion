package search

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

//go:generate go run ../../cmd/mkunion

var (
	ErrEmptyCommand   = fmt.Errorf("empty command")
	ErrDocumentExists = fmt.Errorf("document already exists")
)

type Tokenizer interface {
	Tokenize(text string) []string
}

func NewSimpleTokenizer() *SimpleTokenizer {
	return &SimpleTokenizer{}
}

type SimpleTokenizer struct{}

func (s *SimpleTokenizer) Tokenize(text string) []string {
	return strings.Split(text, " ")
}

var _ Tokenizer = (*SimpleTokenizer)(nil)

func NewIndex(tokenizer Tokenizer) *Index {
	return &Index{
		tokenizer:     tokenizer,
		invertedIndex: make(map[Token]map[DocID]struct{}),
		documents:     make(map[DocID]*Doc),
	}
}

type Token = string
type DocID = string

type Doc struct {
	Raw           string
	Tokens        []Token
	TermFrequency map[Token]float64
}

type TermInDoc struct {
	Position int
}

type Index struct {
	tokenizer Tokenizer

	documents     map[DocID]*Doc
	invertedIndex map[Token]map[DocID]struct{}
}

func (i *Index) Add(docID DocID, text string) error {
	if _, ok := i.documents[docID]; ok {
		return fmt.Errorf("search.Index.Add: document %q; %w", docID, ErrDocumentExists)
	}

	tokens := i.tokenizer.Tokenize(text)
	tokensFrequency := make(map[Token]float64)

	for _, token := range tokens {
		if _, ok := tokensFrequency[token]; !ok {
			tokensFrequency[token] = 0
		}

		tokensFrequency[token]++

		if _, ok := i.invertedIndex[token]; !ok {
			i.invertedIndex[token] = make(map[DocID]struct{})
		}

		i.invertedIndex[token][docID] = struct{}{}
	}

	i.documents[docID] = &Doc{
		Raw:           text,
		Tokens:        tokens,
		TermFrequency: tokensFrequency,
	}

	return nil
}

//go:tag mkunion:"SearchCMD"
type (
	Term struct {
		Term string
	}
	Fulltext struct {
		Query string
	}
)

type Hit struct {
	DocID DocID
	Score float64
	Raw   string
}

type SearchResult struct {
	Results []Hit
}

func (i *Index) Search(cmd SearchCMD) (*SearchResult, error) {
	if cmd == nil {
		return nil, ErrEmptyCommand
	}

	return MatchSearchCMDR2(
		cmd,
		func(x *Term) (*SearchResult, error) {
			result := &SearchResult{
				Results: make([]Hit, 0),
			}

			if _, ok := i.invertedIndex[x.Term]; !ok {
				return result, nil
			}

			N := float64(len(i.documents))
			DF := float64(len(i.invertedIndex[x.Term]))
			for docID := range i.invertedIndex[x.Term] {
				// calculate score as TF-IDF
				// T - how many times term appears in document
				T := i.documents[docID].TermFrequency[x.Term]
				// F - how many terms in document
				F := float64(len(i.documents[docID].Tokens))
				TF := T / F

				IDF := math.Log(N) - math.Log(DF)
				TF_IDF := TF * IDF

				result.Results = append(result.Results, Hit{
					DocID: docID,
					Score: TF_IDF,
					Raw:   i.documents[docID].Raw,
				})
			}

			sort.SliceStable(result.Results, func(i, j int) bool {
				return result.Results[i].Score > result.Results[j].Score
			})

			return result, nil
		},
		func(x *Fulltext) (*SearchResult, error) {
			result := &SearchResult{
				Results: make([]Hit, 0),
			}

			queryTokens := i.tokenizer.Tokenize(x.Query)
			// find all documents that contain all query tokens
			// calculate score as TF-IDF
			// for query token match in document token increase TF
			// for each query token increase IDF

			N := float64(len(i.documents))
			vIDF := NewVector(len(queryTokens))
			for _, token := range queryTokens {
				if _, ok := i.invertedIndex[token]; !ok {
					continue
				}

				DF := float64(len(i.invertedIndex[token]))
				IDF := math.Log(N) - math.Log(DF)
				vIDF.Add(IDF)
			}

			documents := make(map[DocID]struct{})
			for _, token := range queryTokens {
				if _, ok := i.invertedIndex[token]; !ok {
					continue
				}

				for docID := range i.invertedIndex[token] {
					documents[docID] = struct{}{}
				}
			}

			for docID := range documents {
				vTF := NewVector(len(queryTokens))
				for _, token := range queryTokens {
					if _, ok := i.invertedIndex[token][docID]; !ok {
						vTF.Add(0)
						continue
					}

					T := i.documents[docID].TermFrequency[token]
					F := float64(len(i.documents[docID].Tokens))
					vTF.Add(T / F)
				}

				vTF_IDF := vIDF.Dot(vTF)
				result.Results = append(result.Results, Hit{
					DocID: docID,
					Score: vTF_IDF.Mean(),
					Raw:   i.documents[docID].Raw,
				})
			}

			sort.SliceStable(result.Results, func(i, j int) bool {
				return result.Results[i].Score > result.Results[j].Score
			})

			return result, nil
		},
	)
}
