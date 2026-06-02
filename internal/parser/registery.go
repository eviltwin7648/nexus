package parser

import "fmt"

type Registry struct {
	parsers map[string]Parser
}

func NewRegistry(parsers ...Parser) *Registry {
	m := make(map[string]Parser)
	for _, p := range parsers {
		m[p.Language()] = p
	}
	return &Registry{
		parsers: m,
	}
}

func (r *Registry) Get(language string) (Parser, error) {
	p, ok := r.parsers[language]
	if !ok {
		return nil, fmt.Errorf("no parser registered for language: %s", language)
	}

	return p, nil
}
