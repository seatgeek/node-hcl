/*
 * Copyright SeatGeek
 * Licensed under the terms of the Apache-2.0 license. See LICENSE file in project root for terms.
 */
package hcl

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func formatBlockKey(block *hclwrite.Block) string {
	return block.Type() + "." + strings.Join(block.Labels(), ".")
}

func blockToMap(blocks []*hclwrite.Block) map[string]*hclwrite.Block {
	blockMap := make(map[string]*hclwrite.Block)
	for _, block := range blocks {
		blockKey := formatBlockKey(block)
		blockMap[blockKey] = block
	}
	return blockMap
}

// mergeTokens only merges tokens if they are a map, otherwise defaults to the aTokens
// only merges the top level keys of the map, if it exists the value is overridden
func (m *Merger) mergeTokens(aTokens hclwrite.Tokens, bTokens hclwrite.Tokens) (hclwrite.Tokens, error) {
	if aTokens[0].Type != hclsyntax.TokenOBrace || aTokens[len(aTokens)-1].Type != hclsyntax.TokenCBrace {
		return bTokens, nil
	}

	if bTokens[0].Type != hclsyntax.TokenOBrace || bTokens[len(bTokens)-1].Type != hclsyntax.TokenCBrace {
		return bTokens, nil
	}

	aMap, err := objectForTokensMap(aTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize tokens: %w", err)
	}

	bMap, err := objectForTokensMap(bTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize tokens: %w", err)
	}

	outMap := make(map[string]hclwrite.ObjectAttrTokens)
	// this merges the top layer of the map, where nested maps are overwritten
	maps.Copy(outMap, aMap)
	maps.Copy(outMap, bMap)

	var values []hclwrite.ObjectAttrTokens

	// sort the keys to ensure consistent ordering
	keys := make([]string, 0, len(outMap))
	for key := range outMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		values = append(values, outMap[key])
	}

	return hclwrite.TokensForObject(values), nil
}

// mergeAttrs merges two blocks' attributes together. Attributes are composed of hclTokens
// and are identified by their key, e.g.
// key = value
// or key = { ... }
func (m *Merger) mergeAttrs(aAttr map[string]*hclwrite.Attribute, bAttr map[string]*hclwrite.Attribute) map[string]hclwrite.Tokens {
	outAttr := make(map[string]hclwrite.Tokens)

	for key, aValue := range aAttr {
		bValue, found := bAttr[key]

		aAttrTokens := aValue.Expr().BuildTokens(nil)

		if found && m.options.MergeMapKeys {
			bAttrTokens := bValue.Expr().BuildTokens(nil)
			// attempt to merge the value, which are a list of attributes broken up into hclTokens
			mergedTokens, err := m.mergeTokens(aAttrTokens, bAttrTokens)
			if err != nil {
				// if there was an error merging the tokens, default to the bAttrTokens
				outAttr[key] = bAttrTokens
				continue
			}

			outAttr[key] = mergedTokens
		} else if found {
			// if the key is found in both attributes, default to the bAttrTokens
			bAttrTokens := bValue.Expr().BuildTokens(nil)
			outAttr[key] = bAttrTokens
		} else {
			outAttr[key] = aAttrTokens
		}
	}

	// add any attributes that are in bAttr but not in aAttr
	for key, bValue := range bAttr {
		_, found := aAttr[key]

		if !found {
			bAttrTokens := bValue.Expr().BuildTokens(nil)
			outAttr[key] = bAttrTokens
		}
	}

	return outAttr
}

// objectForTokensMap is the inverse of hclwrite.TokensForObject, only merges the top level keys, but not the values of the keys.
// if a value exists, it is overridden
func objectForTokensMap(tokens hclwrite.Tokens) (map[string]hclwrite.ObjectAttrTokens, error) {
	if len(tokens) < 2 || tokens[0].Type != hclsyntax.TokenOBrace || tokens[len(tokens)-1].Type != hclsyntax.TokenCBrace {
		return nil, fmt.Errorf("tokens are not a valid object")
	}

	result := make(map[string]hclwrite.ObjectAttrTokens)
	var currentKey string                  // used for the result map
	var currentKeyTokens hclwrite.Tokens   // used for the ObjectAttrTokens key
	var currentValueTokens hclwrite.Tokens // used for the ObjectAttrTokens value
	var inValue bool                       // flag to determine if we are in the value part of the tokens when parsing

	i := 1                  // start after the opening brace
	for i < len(tokens)-1 { // skip the closing brace
		token := tokens[i]

		switch token.Type {
		case hclsyntax.TokenIdent, hclsyntax.TokenQuotedLit:
			if inValue {
				// set the value if in the value
				currentValueTokens = append(currentValueTokens, token)
			} else {
				// set the key
				currentKey = string(token.Bytes)
				currentKeyTokens = append(currentKeyTokens, token)
			}

		case hclsyntax.TokenEqual:
			// flag that we are in the value part of the tokens
			inValue = true

		case hclsyntax.TokenOBrace, hclsyntax.TokenOBrack:
			// find the closing token for the look ahead
			cToken := hclsyntax.TokenCBrace
			if token.Type == hclsyntax.TokenOBrack {
				cToken = hclsyntax.TokenCBrack
			}

			// look ahead to find the end index of map/array
			unclosedTokens := 1
			endIndex := -1
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type == token.Type {
					unclosedTokens++
				} else if tokens[j].Type == cToken {
					unclosedTokens--
				}
				if unclosedTokens == 0 {
					endIndex = j
					break
				}
			}

			if endIndex == -1 {
				return nil, fmt.Errorf("failed to find closing token")
			}

			// include the tokens for the map/array in the current value
			currentValueTokens = append(currentValueTokens, tokens[i:endIndex+1]...)
			i = endIndex

		case hclsyntax.TokenNewline, hclsyntax.TokenComma:
			// if at the end of the value, add the key and value to the result map
			if inValue {
				result[currentKey] = hclwrite.ObjectAttrTokens{
					Name:  currentKeyTokens,
					Value: currentValueTokens,
				}

				// reset the current key and value tokens to parse the next attribute
				currentKey = ""
				currentKeyTokens = hclwrite.Tokens{}
				currentValueTokens = hclwrite.Tokens{}
				inValue = false
			}

		default:
			if inValue {
				// add tokens to the value until we hit the end of the value (comma or newline)
				currentValueTokens = append(currentValueTokens, token)
			} else {
				// add tokens to the key until we hit the end of the key (equal sign)
				currentKeyTokens = append(currentKeyTokens, token)
			}
		}

		i++
	}

	// add the last attribute found to the result map
	if len(currentKeyTokens) > 0 && len(currentValueTokens) > 0 {
		result[currentKey] = hclwrite.ObjectAttrTokens{
			Name:  currentKeyTokens,
			Value: currentValueTokens,
		}
	}

	return result, nil
}

// mergeFiles merges two HCL files together
func (m *Merger) mergeFiles(aFile *hclwrite.File, bFile *hclwrite.File) *hclwrite.File {
	out := hclwrite.NewFile()
	outBlocks := m.mergeBlocks(aFile.Body().Blocks(), bFile.Body().Blocks())

	lastIndex := len(outBlocks) - 1

	for i, block := range outBlocks {
		out.Body().AppendBlock(block)
		out.Body().AppendNewline()

		// append extra newline for spacing between blocks, but not at the EOF
		if i < lastIndex {
			out.Body().AppendNewline()
		}
	}

	return out
}

// mergeBlocks merges two blocks together, a block is identified by its type and labels, e.g.
// type "label" { ... }
// or type { ... }
func (m *Merger) mergeBlocks(aBlocks []*hclwrite.Block, bBlocks []*hclwrite.Block) []*hclwrite.Block {
	outBlocks := make([]*hclwrite.Block, 0)
	aBlockMap := blockToMap(aBlocks)
	bBlockMap := blockToMap(bBlocks)

	for _, aBlock := range aBlocks {
		blockKey := formatBlockKey(aBlock)
		outBlock := aBlock
		bBlock, found := bBlockMap[blockKey]

		if found {
			// override outBlock with the new block to merge the two blocks into
			outBlock = hclwrite.NewBlock(aBlock.Type(), aBlock.Labels())

			// merge block attributes
			outAttributes := m.mergeAttrs(aBlock.Body().Attributes(), bBlock.Body().Attributes())
			// sort the keys to ensure consistent ordering
			keys := make([]string, 0, len(outAttributes))
			for key := range outAttributes {
				keys = append(keys, key)
			}

			sort.Strings(keys)

			for _, key := range keys {
				outBlock.Body().SetAttributeRaw(key, outAttributes[key])
			}

			// recursively merge nested blocks
			aNestedBlocks := aBlock.Body().Blocks()
			bNestedBlocks := bBlock.Body().Blocks()
			outNestedBlocks := m.mergeBlocks(aNestedBlocks, bNestedBlocks)

			for _, nestedBlock := range outNestedBlocks {
				outBlock.Body().AppendNewline()
				outBlock.Body().AppendBlock(nestedBlock)
			}
		}

		outBlocks = append(outBlocks, outBlock)
	}

	for _, bBlock := range bBlocks {
		blockKey := formatBlockKey(bBlock)
		_, found := aBlockMap[blockKey]

		if !found {
			// append any target blocks that were not in the source
			outBlocks = append(outBlocks, bBlock)
		}
	}

	return outBlocks
}

func parseBytes(bytes []byte) (*hclwrite.File, error) {
	sourceHclFile, d := hclwrite.ParseConfig(bytes, "", hcl.InitialPos)
	if d.HasErrors() {
		return nil, fmt.Errorf("error parsing hcl file: %v", d.Error())
	}

	return sourceHclFile, nil
}

// MergeOptions are the options for merging two HCL strings
type MergeOptions struct {
	// MergeMapKeys merges the keys of maps together, note this does not merge the values of the keys. If
	// unset, the keys of the second map will override the keys of the first map.
	MergeMapKeys bool
}

type merger interface {
	Merge(a string, b string) (string, error)
}

var _ merger = &Merger{}

// NewMerger creates a new Merger with the provided options
func NewMerger(options *MergeOptions) *Merger {
	if options == nil {
		options = &MergeOptions{}
	}

	return &Merger{
		options: options,
	}
}

// Merger is the struct that merges two HCL strings together
type Merger struct {
	options *MergeOptions
}

// Merge merges two HCL strings together
func (m *Merger) Merge(a string, b string) (string, error) {
	aBytes := []byte(a)
	bBytes := []byte(b)

	// safe parse the HCL files
	aFile, err := parseBytes(aBytes)
	if err != nil {
		return "", err
	}

	bFile, err := parseBytes(bBytes)
	if err != nil {
		return "", err
	}

	// merge the blocks and attributes from the HCL files
	outFile := m.mergeFiles(aFile, bFile)
	outFileFormatted := hclwrite.Format(outFile.Bytes())

	return string(outFileFormatted), nil
}
