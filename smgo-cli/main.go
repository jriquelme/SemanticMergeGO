package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/jriquelme/SemanticMergeGO/smgo"
	"gopkg.in/yaml.v2"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalln("invalid arguments: use smgo-cli shell <flag file path>")
	}
	if os.Args[1] != "shell" {
		log.Fatalln("invalid arguments: use smgo-cli shell <flag file path>")
	}
	flagFilePath := os.Args[2]
	flagFile, err := os.Create(flagFilePath)
	if err != nil {
		log.Fatalf("error creating flag file: %s", err)
	}
	_, err = flagFile.Write([]byte{1})
	if err != nil {
		log.Fatalf("error writting to flag file: %s", err)
	}
	err = flagFile.Close()
	if err != nil {
		log.Fatalf("error closing flag file: %s", err)
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		srcOrEnd := scanner.Text()
		if srcOrEnd == "end" {
			break
		}
		if !scanner.Scan() {
			log.Fatalf("unexpected EOF: %s", scanner.Err())
		}
		encoding := scanner.Text()
		if !scanner.Scan() {
			log.Fatalf("unexpected EOF: %s", scanner.Err())
		}
		output := scanner.Text()

		err := parse(srcOrEnd, encoding, output)
		if err != nil {
			fmt.Println("KO")
		} else {
			fmt.Println("OK")
		}
	}
}

func parse(src, encoding, output string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dtFile, err := smgo.Parse(srcFile, encoding)
	if err != nil {
		return err
	}
	outputFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	yamlFile := toFile(dtFile)
	yamlFile.Name = src

	yamlEncoder := yaml.NewEncoder(outputFile)
	defer yamlEncoder.Close()
	return yamlEncoder.Encode(yamlFile)
}
