package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type WordDictionary struct {
	wordMetaMap   map[string]DictionaryEntryMeta
	currentOffset int64
}

const FilePath = "./word_dictionary"

var file *os.File

func init() {
	var err error
	file, err = os.OpenFile(FilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		log.Fatalf("Error occurred while opening the file for writing")
	}
}

func NewWordDictionary() *WordDictionary {
	info, err := file.Stat()
	if err != nil {
		// handle error
	}
	currentOffset := info.Size()
	return &WordDictionary{
		wordMetaMap:   make(map[string]DictionaryEntryMeta),
		currentOffset: currentOffset,
	}
}

type DictionaryEntryMeta struct {
	offset    int64
	entrySize int
}

// Assumptions:
// Word size max - 8 bits - max 2^8 = 256
// Meaning size max - 24 bits - max 2^24
// Avg size of words in dictionary - 4.7
// Total words - 170000
// Max size of overall file - 1 TB
// Entry is stored as [word size, meaning size, word, meaning]
func (w *WordDictionary) Insert(word, meaning string) error {
	// In case a duplicate arises, don't do anything for now
	if _, ok := w.wordMetaMap[word]; !ok {
		wordBytes := []byte(word)
		meaningBytes := []byte(meaning)

		wordLen := len(wordBytes)
		meaningLen := len(meaningBytes)

		// 1 byte for word length
		wordLenByte := byte(wordLen)

		// 3 bytes for meaning length
		meaningLenBytes := [3]byte{
			byte(meaningLen >> 16), // highest 8 bytes
			byte(meaningLen >> 8),  // middle 8 bytes
			byte(meaningLen),       // lowest 8 bytes
		}
		totalLen := 4 + wordLen + meaningLen
		result := make([]byte, totalLen)
		result[0] = wordLenByte
		copy(result[1:4], meaningLenBytes[:])
		copy(result[4:4+wordLen], wordBytes)
		copy(result[4+wordLen:], meaningBytes)

		_, err := file.Write(result)
		if err != nil {
			fmt.Printf("Error occurred while writing to file: %v\n", err)
			return err
		}

		w.wordMetaMap[word] = DictionaryEntryMeta{
			offset:    w.currentOffset,
			entrySize: totalLen,
		}

		w.currentOffset += int64(totalLen)
	}

	return nil
}

// Implementing the merge functionality for handling bulk changes coming via change log.
// Initially assuming the change log file would also contain changes in the same format as that of the word file
func (w *WordDictionary) Import(changeLogFilePath string) error {
	changeLogFile, err := os.OpenFile(changeLogFilePath, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("error opening change log: %w", err)
	}
	defer changeLogFile.Close()

	info, err := changeLogFile.Stat()
	if err != nil {
		return fmt.Errorf("error stating change log: %w", err)
	}

	tempFilePath := FilePath + "_temp"
	tempFile, err := os.OpenFile(tempFilePath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0777)
	if err != nil {
		return fmt.Errorf("error creating temp file: %w", err)
	}
	defer tempFile.Close()

	tempMetaMap := make(map[string]DictionaryEntryMeta)
	var tempFileOffset int64 = 0

	currentFileOffset := int64(0)
	changeLogFileOffset := int64(0)

	for currentFileOffset < w.currentOffset && changeLogFileOffset < info.Size() {
		// Read current entry from current file
		curLenBytes := make([]byte, 4)
		if _, err = file.ReadAt(curLenBytes, currentFileOffset); err != nil {
			return fmt.Errorf("reading current file length bytes: %w", err)
		}
		curWordLen := int(curLenBytes[0])
		curMeaningLen := int(curLenBytes[1])<<16 | int(curLenBytes[2])<<8 | int(curLenBytes[3])
		curEntrySize := 4 + curWordLen + curMeaningLen

		curPayload := make([]byte, curWordLen+curMeaningLen)
		if _, err = file.ReadAt(curPayload, currentFileOffset+4); err != nil {
			return fmt.Errorf("reading current file payload: %w", err)
		}
		curWord := string(curPayload[:curWordLen])

		// Read change log entry
		chLenBytes := make([]byte, 4)
		if _, err = changeLogFile.ReadAt(chLenBytes, changeLogFileOffset); err != nil {
			return fmt.Errorf("reading change log length bytes: %w", err)
		}

		chWordLen := int(chLenBytes[0])
		chMeaningLen := int(chLenBytes[1])<<16 | int(chLenBytes[2])<<8 | int(chLenBytes[3])
		chEntrySize := 4 + chWordLen + chMeaningLen

		chPayload := make([]byte, chWordLen+chMeaningLen)
		if _, err = changeLogFile.ReadAt(chPayload, changeLogFileOffset+4); err != nil {
			return fmt.Errorf("reading change log payload: %w", err)
		}
		chWord := string(chPayload[:chWordLen])

		cmp := strings.Compare(curWord, chWord)

		var entryBytes []byte
		var entryWord string
		var entrySize int
		// Merge entries based on lexicographical order
		if cmp == 0 {
			// If same word, choose meaning from change log
			entrySize = chEntrySize
			entryBytes = make([]byte, entrySize)
			entryBytes[0] = chLenBytes[0]
			copy(entryBytes[1:4], chLenBytes[1:4])
			copy(entryBytes[4:], chPayload)
			entryWord = chWord

			currentFileOffset += int64(curEntrySize)
			changeLogFileOffset += int64(chEntrySize)

		} else if cmp < 0 {
			// Current word comes first
			entrySize = curEntrySize
			entryBytes = make([]byte, entrySize)
			entryBytes[0] = curLenBytes[0]
			copy(entryBytes[1:4], curLenBytes[1:4])
			copy(entryBytes[4:], curPayload)
			entryWord = curWord

			currentFileOffset += int64(curEntrySize)

		} else {
			// Change log word comes first
			entrySize = chEntrySize
			entryBytes = make([]byte, entrySize)
			entryBytes[0] = chLenBytes[0]
			copy(entryBytes[1:4], chLenBytes[1:4])
			copy(entryBytes[4:], chPayload)
			entryWord = chWord

			changeLogFileOffset += int64(chEntrySize)
		}

		n, err := tempFile.Write(entryBytes)
		if err != nil {
			return fmt.Errorf("writing to temp file: %w", err)
		}
		if n != entrySize {
			return fmt.Errorf("incomplete write to temp file")
		}

		tempMetaMap[entryWord] = DictionaryEntryMeta{
			offset:    tempFileOffset,
			entrySize: entrySize,
		}
		tempFileOffset += int64(entrySize)
	}

	// Append remaining entries from current file
	for currentFileOffset < w.currentOffset {
		// Read length prefix of 4 bytes for one entry
		lenBytes := make([]byte, 4)
		if _, err := file.ReadAt(lenBytes, currentFileOffset); err != nil {
			return fmt.Errorf("reading current file length bytes: %w", err)
		}
		wordLen := int(lenBytes[0])
		meaningLen := int(lenBytes[1])<<16 | int(lenBytes[2])<<8 | int(lenBytes[3])
		entrySize := 4 + wordLen + meaningLen

		// Read entire entry bytes
		buf := make([]byte, entrySize)
		if _, err := file.ReadAt(buf, currentFileOffset); err != nil {
			return fmt.Errorf("reading current file entry bytes: %w", err)
		}

		wordBytes := buf[4 : 4+wordLen]
		word := string(wordBytes)

		// Write entry to temp file
		n, err := tempFile.Write(buf)
		if err != nil {
			return fmt.Errorf("writing current file entry to temp file: %w", err)
		}
		if n != entrySize {
			return fmt.Errorf("incomplete write of current file entry")
		}

		// Update meta map
		tempMetaMap[word] = DictionaryEntryMeta{
			offset:    tempFileOffset,
			entrySize: entrySize,
		}

		tempFileOffset += int64(entrySize)
		currentFileOffset += int64(entrySize)
	}

	// Append remaining entries from change log file
	for changeLogFileOffset < info.Size() {
		// Read length prefix for one entry (4 bytes)
		lenBytes := make([]byte, 4)
		if _, err := changeLogFile.ReadAt(lenBytes, changeLogFileOffset); err != nil {
			return fmt.Errorf("reading length bytes: %w", err)
		}

		wordLen := int(lenBytes[0])
		meaningLen := int(lenBytes[1])<<16 | int(lenBytes[2])<<8 | int(lenBytes[3])
		entrySize := 4 + wordLen + meaningLen

		buf := make([]byte, entrySize)
		if _, err := changeLogFile.ReadAt(buf, changeLogFileOffset); err != nil {
			return fmt.Errorf("reading entry bytes: %w", err)
		}

		wordBytes := buf[4 : 4+wordLen]
		word := string(wordBytes)

		// Update the meta map
		tempMetaMap[word] = DictionaryEntryMeta{
			offset:    tempFileOffset,
			entrySize: entrySize,
		}

		// Write this entry to the temp file
		n, err := tempFile.Write(buf)
		if err != nil {
			return fmt.Errorf("writing temp file: %w", err)
		}
		if n != entrySize {
			return fmt.Errorf("incomplete write to temp file")
		}

		// Advance offsets
		tempFileOffset += int64(entrySize)
		changeLogFileOffset += int64(entrySize)
	}

	tempFile.Sync()
	tempFile.Close()

	file.Close() // Close the old file handle

	// Rename temp file to original file (atomic replace)
	if err := os.Rename(tempFilePath, FilePath); err != nil {
		return fmt.Errorf("renaming temp file failed: %w", err)
	}

	// Reopen new file handle
	file, err = os.OpenFile(FilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("reopening file failed: %w", err)
	}

	// Swap the map and current offset atomically for this instance
	w.wordMetaMap = tempMetaMap
	w.currentOffset = tempFileOffset

	return nil
}

// Given a word - returns the corresponding meaning in the word dictionary
// Steps involved:
// 1. Get the entry in the wordMetaMap
// 2. If the entry doesn't exist, then return empty value
// 3. If the entry exists, then read the file and then go to the offset and get entrySize number of bytes into a buffer
func (w *WordDictionary) GetMeaning(word string) (string, error) {
	meta, ok := w.wordMetaMap[word]
	if !ok {
		return "", nil
	}

	offset := meta.offset
	entrySize := meta.entrySize

	byteArr := make([]byte, entrySize)

	_, err := file.ReadAt(byteArr, offset)
	if err != nil {
		return "", nil
	}

	// Read the 2nd to 4th bytes to know about the length of the meaning
	wordLenInt := int(byteArr[0])
	meaningLen := byteArr[1:4]
	meaningLenInt := int(meaningLen[0])<<16 | int(meaningLen[1])<<8 | int(meaningLen[2])

	wordBytes := byteArr[4 : 4+wordLenInt]
	meaningBytes := byteArr[4+wordLenInt : 4+wordLenInt+meaningLenInt]

	wordStr := string(wordBytes)
	meaningStr := string(meaningBytes)

	fmt.Printf("Meaning of %s: %s\n", wordStr, meaningStr)

	return meaningStr, nil
}

func main() {
	dict := NewWordDictionary()
	dict.Insert("Apple", "A fruit")
	dict.Insert("Ball", "A Ball")
	dict.Insert("Cat", "A Cat")

	dict.Import("./change_log")

	file.Sync()
	file.Close()
}
