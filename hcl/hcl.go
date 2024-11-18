/*
 * Copyright SeatGeek
 * Licensed under the terms of the Apache-2.0 license. See LICENSE file in project root for terms.
 */
package hcl

import (
	"fmt"
	"maps"
	"sort"
	"strconv"
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

// mergeTokens only merges tokens if they are a map or slice, otherwise defaults to the aTokens
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

	outMap := make(map[string]interface{})
	// TODO: support merging nested maps
	// this merges the top layer of the map, where nested maps are overwritten
	maps.Copy(outMap, aMap)
	maps.Copy(outMap, bMap)

	return convertMapToTokens(outMap), nil
}

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

		// merge tokens
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

func convertTokensToMap(tokens hclwrite.Tokens) (map[string]interface{}, error) {
	if tokens[0].Type != hclsyntax.TokenOBrace || tokens[len(tokens)-1].Type != hclsyntax.TokenCBrace {
		return nil, fmt.Errorf("tokens are not a map")
	}

	result := make(map[string]interface{})
	var currentKey string

	// offset to skip the opening and closing braces
	for i := 1; i < len(tokens)-1; i++ {
		token := tokens[i]

		switch token.Type {
		case hclsyntax.TokenIdent:
			currentKey = string(token.Bytes)

		case hclsyntax.TokenOBrace:
			unclosedBraces := 1
			endIndex := -1

			// find the closing brace
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j].Type == hclsyntax.TokenOBrace {
					unclosedBraces++
				} else if tokens[j].Type == hclsyntax.TokenCBrace {
					unclosedBraces--
				}

				if unclosedBraces == 0 {
					endIndex = j
					break
				}
			}

			if endIndex == -1 {
				return nil, fmt.Errorf("failed to find closing brace")
			}

			// get the index of the OBrace and CBrace and call recursively
			subTokens := tokens[i : endIndex+1]
			subMap, err := convertTokensToMap(subTokens)
			if err != nil {
				return nil, fmt.Errorf("failed to parse map: %w", err)
			}

			result[currentKey] = subMap
			currentKey = ""
			i = endIndex

		// parse quoted string values
		case hclsyntax.TokenQuotedLit:
			value := string(token.Bytes)
			if len(currentKey) == 0 {
				currentKey = value
				break
			}

			result[currentKey] = value
			currentKey = ""

		// parse string values
		case hclsyntax.TokenStringLit:
			value := string(token.Bytes)
			if len(currentKey) == 0 {
				currentKey = value
				break
			}

			result[currentKey] = value
			currentKey = ""

		// parse int values
		case hclsyntax.TokenNumberLit:
			value, err := strconv.Atoi(string(token.Bytes))
			if err != nil {
				return nil, fmt.Errorf("failed to parse int: %w", err)
			}

			result[currentKey] = value
			currentKey = ""

		// ignore remaining tokens
		default:

		}
	}

	if currentKey != "" {
		return nil, fmt.Errorf("incomplete key-value pair, key: %s", currentKey)
	}

	return result, nil
}

func convertMapToTokens(input map[string]interface{}) hclwrite.Tokens {
	tokens := hclwrite.Tokens{}

	// add opening brace for the map
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte("{"),
	})

	// add newline
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenNewline,
		Bytes: []byte("\n"),
	})

	// sort keys for consistent ordering
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	// iterate through the map
	for _, key := range keys {
		value := input[key]

		// add the map key
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenIdent,
			Bytes: []byte(key),
		})

		// add the equal sign
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenEqual,
			Bytes: []byte("="),
		})

		// convert type to the proper token type
		switch v := value.(type) {
		case string:
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenQuotedLit,
				Bytes: []byte(fmt.Sprintf("%q", v)), // Wrap in quotes
			})

		case int:
			tokens = append(tokens, &hclwrite.Token{
				Type:  hclsyntax.TokenNumberLit,
				Bytes: []byte(fmt.Sprintf("%d", v)),
			})

		case map[string]interface{}:
			nestedTokens := convertMapToTokens(v)
			tokens = append(tokens, nestedTokens...)

		default:
			// ignore unsupported types
		}

		// add newline
		tokens = append(tokens, &hclwrite.Token{
			Type:  hclsyntax.TokenNewline,
			Bytes: []byte("\n"),
		})
	}

	// add closing brace for the map
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenCBrace,
		Bytes: []byte("}"),
	})

	return tokens
}

func merge(aFile *hclwrite.File, bFile *hclwrite.File) *hclwrite.File {
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

	// merge the blocks from the HCL files
	outFile := merge(aFile, bFile)
	outFileFormatted := hclwrite.Format(outFile.Bytes())

	return string(outFileFormatted), nil
}
