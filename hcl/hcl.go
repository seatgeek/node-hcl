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
func mergeTokens(aTokens hclwrite.Tokens, bTokens hclwrite.Tokens) (hclwrite.Tokens, error) {
	if aTokens[0].Type != hclsyntax.TokenOBrace || aTokens[len(aTokens)-1].Type != hclsyntax.TokenCBrace {
		return aTokens, nil
	}

	if bTokens[0].Type != hclsyntax.TokenOBrace || bTokens[len(bTokens)-1].Type != hclsyntax.TokenCBrace {
		return aTokens, nil
	}

	aMap, err := convertTokensToMap(aTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize tokens: %w", err)
	}

	bMap, err := convertTokensToMap(bTokens)
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
func mergeAttrs(aBlock *hclwrite.Block, bBlock *hclwrite.Block) {
	attributes := aBlock.Body().Attributes()

	// sort the attributes to ensure consistent ordering
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		aAttrTokens := attributes[key].Expr().BuildTokens(nil)
		bAttr := bBlock.Body().GetAttribute(key)

		if bAttr == nil {
			bBlock.Body().SetAttributeRaw(key, aAttrTokens)
			continue
		}

		// merge the value, which are a list of attributes broken up into hclTokens
		// TODO: gate merging tokens behind an option flag
		mergedTokens, err := mergeTokens(aAttrTokens, bAttr.Expr().BuildTokens(nil))
		if err != nil {
			// if there was an error merging the tokens, default to the aAttr
			bBlock.Body().SetAttributeRaw(key, aAttrTokens)
			continue
		}

		bBlock.Body().SetAttributeRaw(key, mergedTokens)
	}
}

func convertTokensToMap(tokens hclwrite.Tokens) (map[string]hclwrite.ObjectAttrTokens, error) {
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
func mergeFiles(aFile *hclwrite.File, bFile *hclwrite.File) *hclwrite.File {
	out := hclwrite.NewFile()
	outBlocks := mergeBlocks(aFile.Body().Blocks(), bFile.Body().Blocks())

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
func mergeBlocks(aBlocks []*hclwrite.Block, bBlocks []*hclwrite.Block) []*hclwrite.Block {
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

			// set block attributes of the new block
			mergeAttrs(aBlock, outBlock)
			mergeAttrs(bBlock, outBlock)

			// recursively merge nested blocks
			aNestedBlocks := aBlock.Body().Blocks()
			bNestedBlocks := bBlock.Body().Blocks()
			outNestedBlocks := mergeBlocks(aNestedBlocks, bNestedBlocks)

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

// Merge merges two HCL strings together
func Merge(a string, b string) (string, error) {
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
	outFile := mergeFiles(aFile, bFile)
	outFileFormatted := hclwrite.Format(outFile.Bytes())

	return string(outFileFormatted), nil
}
