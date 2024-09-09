/*
 * Copyright SeatGeek
 * Licensed under the terms of the Apache-2.0 license. See LICENSE file in project root for terms.
 */
package hcl

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
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

func setAttrs(sourceBlock *hclwrite.Block, targetBlock *hclwrite.Block) {
	attributes := sourceBlock.Body().Attributes()

	// sort the attributes to ensure consistent ordering
	keys := make([]string, 0, len(attributes))
	for key := range attributes {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		targetBlock.Body().SetAttributeRaw(key, attributes[key].Expr().BuildTokens(nil))
	}
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
			setAttrs(aBlock, outBlock)
			setAttrs(bBlock, outBlock)

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

func parseFileSafe(path string) (*hclwrite.File, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return hclwrite.NewFile(), nil
		}
		return nil, err
	}
	defer file.Close()

	return parseFile(path)
}

func parseFile(path string) (*hclwrite.File, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return parseBytes(bytes)
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
	out := merge(aFile, bFile)

	// out to buffer
	var buf bytes.Buffer
	_, err = out.WriteTo(&buf)
	if err != nil {
		return "", fmt.Errorf("error writing HCL to file: %w", err)
	}

	return buf.String(), nil
}

func MergeWrite(a string, b string, outPath string) error {
	aBytes := []byte(a)
	bBytes := []byte(b)

	// safe parse the HCL files
	aFile, err := parseBytes(aBytes)
	if err != nil {
		return err
	}

	bFile, err := parseBytes(bBytes)
	if err != nil {
		return err
	}

	// merge the blocks from the HCL files
	out := merge(aFile, bFile)

	// create the output file
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer outFile.Close()

	// write the HCL content to the file
	_, err = out.WriteTo(outFile)
	if err != nil {
		return fmt.Errorf("error writing HCL to file: %w", err)
	}

	return nil
}

func MergeFiles(aPath string, bPath string) (string, error) {
	// safe parse the HCL files
	aFile, err := parseFileSafe(aPath)
	if err != nil {
		return "", err
	}

	bFile, err := parseFileSafe(bPath)
	if err != nil {
		return "", err
	}

	// merge the blocks from the HCL files
	out := merge(aFile, bFile)

	// out to buffer
	var buf bytes.Buffer
	_, err = out.WriteTo(&buf)
	if err != nil {
		return "", fmt.Errorf("error writing HCL to file: %w", err)
	}

	return buf.String(), nil
}

func MergeFilesWrite(aPath string, bPath string, outPath string) error {
	// safe parse the HCL files
	aFile, err := parseFileSafe(aPath)
	if err != nil {
		return err
	}

	bFile, err := parseFileSafe(bPath)
	if err != nil {
		return err
	}

	// merge the blocks from the HCL files
	out := merge(aFile, bFile)

	// create the output file
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer outFile.Close()

	// write the HCL content to the file
	_, err = out.WriteTo(outFile)
	if err != nil {
		return fmt.Errorf("error writing HCL to file: %w", err)
	}

	return nil
}
