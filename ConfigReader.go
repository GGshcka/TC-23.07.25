package main

import (
	"encoding/xml"
	"fmt"
	"os"
)

type Data struct {
	Port           Port           `xml:"port"`
	FileExtensions FileExtensions `xml:"fileExtensions"`
}

type Port struct {
	Value int `xml:"value,attr"`
}

type FileExtensions struct {
	Extensions []FileExtension `xml:"extension"`
}

type FileExtension struct {
	Value string `xml:"value,attr"`
}

func NewConfigReader() (Data, error) {
	f, err := os.Open("config.xml")
	if err != nil {
		fmt.Println("Error file opening:", err)
		return Data{}, err
	}
	defer f.Close()

	var data Data
	decoder := xml.NewDecoder(f)

	err = decoder.Decode(&data)
	if err != nil {
		fmt.Println("Error while decoding XML:", err)
		return Data{}, err
	}
	return data, nil
}
