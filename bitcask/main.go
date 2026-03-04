package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sync"
	"time"
)

type KeyDirEntry struct {
	filePath        string
	fileEntryOffset int64
	valueSize       uint32
}

type BitCask struct {
	fileBasePath     string
	currOffset       int64
	filePtrs         map[string]*os.File
	keyDir           map[string]KeyDirEntry
	mtx              sync.RWMutex
	fileCnt          int
	maxFileSizeBytes int64
	mergeFileCnt     int
	mergeFileOffset  int64
}

type mergeUpdate struct {
	originalFilePath string
	originalOffset   int64
	originalValSize  uint32
	newEntry         KeyDirEntry
}

const headerSize = 17 // crc(4) + tstamp(4) + ksz(4) + vsz(4) + flag(1)

func (b *BitCask) put(key, value []byte, flag byte) error {

	keyLen := len(key)
	valueLen := len(value)
	timestamp := uint32(time.Now().Unix())

	entrySize := headerSize + keyLen + valueLen
	buf := make([]byte, entrySize)
	binary.LittleEndian.PutUint32(buf[4:8], timestamp)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(keyLen))
	binary.LittleEndian.PutUint32(buf[12:16], uint32(valueLen))
	buf[16] = flag
	copy(buf[headerSize:], key)
	copy(buf[headerSize+keyLen:], value)

	checkSum := crc32.ChecksumIEEE(buf[4:])
	binary.LittleEndian.PutUint32(buf[0:4], checkSum)

	keyStr := string(key)

	b.mtx.Lock()
	defer b.mtx.Unlock()
	filePath := fmt.Sprintf("%s%d.data", b.fileBasePath, b.fileCnt)
	if len(b.filePtrs) == 0 {
		filePtr, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("Error occurred while creating filePtr:", err)
			return err
		}
		b.filePtrs[filePath] = filePtr
	}

	numBytes, err := b.filePtrs[filePath].WriteAt(buf, b.currOffset)
	if err != nil {
		fmt.Printf("Error occurred while writing to file for key: %s: %v\n", keyStr, err)
		return err
	}
	if numBytes != entrySize {
		fmt.Printf("Incomplete entry written for key: %s\n", keyStr)
		return errors.New("Incomplete entry written")
	}

	if flag == 0 {
		b.keyDir[keyStr] = KeyDirEntry{
			filePath,
			b.currOffset,
			uint32(valueLen),
		}
	} else {
		delete(b.keyDir, keyStr)
	}

	b.currOffset += int64(entrySize)

	if b.currOffset >= b.maxFileSizeBytes {
		b.fileCnt++
		newFilePath := fmt.Sprintf("%s%d.data", b.fileBasePath, b.fileCnt)
		filePtr, err := os.OpenFile(newFilePath, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			fmt.Println("Error occurred while creating filePtr:", err)
			return err
		}
		b.filePtrs[newFilePath] = filePtr
		b.currOffset = 0
	}

	return nil
}

func (b *BitCask) Put(key, value []byte) error {
	return b.put(key, value, 0)
}

func (b *BitCask) Delete(key []byte) error {
	return b.put(key, nil, 1)
}

func (b *BitCask) Get(key []byte) ([]byte, error) {
	b.mtx.RLock()
	keyDirEntry, ok := b.keyDir[string(key)]

	if !ok {
		fmt.Println("Key doesn't exist in the index")
		b.mtx.RUnlock()
		return nil, errors.New("Key not found")
	}

	filePtr := b.filePtrs[keyDirEntry.filePath]
	b.mtx.RUnlock()

	keyLen := len(key)

	buf := make([]byte, headerSize+keyLen+int(keyDirEntry.valueSize))
	_, err := filePtr.ReadAt(buf, keyDirEntry.fileEntryOffset)
	if err != nil {
		fmt.Println("Error occurred while reading from offset:", err)
		return nil, err
	}

	crc := binary.LittleEndian.Uint32(buf[0:4])
	checkSum := crc32.ChecksumIEEE(buf[4:])
	if checkSum != crc {
		fmt.Println("Checksum doesn't match for key")
		return nil, errors.New("Corrupted entry found")
	}

	fmt.Println("Returned value: ", string(buf[headerSize+keyLen:]))
	return buf[headerSize+keyLen:], nil
}

func (b *BitCask) Merge() {

	b.mtx.Lock()
	activeFilePath := fmt.Sprintf("%s%d.data", b.fileBasePath, b.fileCnt)
	mergeFilePath := fmt.Sprintf("%smerge-%d.data", b.fileBasePath, b.mergeFileCnt)
	mergeFilePtr, err := os.OpenFile(mergeFilePath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Error occurred while creating filePtr:", err)
		b.mtx.Unlock()
		return
	}
	b.filePtrs[mergeFilePath] = mergeFilePtr

	// Snapshot all immutable file paths (everything except active and current merge file)
	var immutableFiles []string
	for path := range b.filePtrs {
		if path != activeFilePath && path != mergeFilePath {
			immutableFiles = append(immutableFiles, path)
		}
	}
	b.mtx.Unlock()

	for _, filePath := range immutableFiles {
		buf := make([]byte, 17)
		b.mtx.RLock()
		filePtr, exists := b.filePtrs[filePath]
		b.mtx.RUnlock()
		if !exists {
			continue
		}
		curOffset := int64(0)
		fileMergedSuccessfully := true
		pendingUpdates := make(map[string]mergeUpdate)
		for {
			_, err := filePtr.ReadAt(buf, curOffset)
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Printf("Error occurred while reading header: %v\n", err)
				fileMergedSuccessfully = false
				break
			}
			keySize := binary.LittleEndian.Uint32(buf[8:12])
			valueSize := binary.LittleEndian.Uint32(buf[12:16])
			isDeleted := buf[16]

			blockSize := int64(17 + keySize + valueSize)
			if isDeleted == 0 {
				buf1 := make([]byte, blockSize)
				_, err = filePtr.ReadAt(buf1, curOffset)
				if err != nil {
					fmt.Printf("Error occurred while reading entry: %v\n", err)
					fileMergedSuccessfully = false
					break
				}

				// CRC verification
				storedCRC := binary.LittleEndian.Uint32(buf1[0:4])
				computedCRC := crc32.ChecksumIEEE(buf1[4:])
				if storedCRC != computedCRC {
					fmt.Printf("Corrupted entry at offset %d in %s, skipping\n", curOffset, filePath)
					curOffset += blockSize
					continue
				}

				key := buf1[17 : 17+keySize]
				keyStr := string(key)

				b.mtx.RLock()
				keyDirEntry, ok := b.keyDir[keyStr]
				b.mtx.RUnlock()
				if ok {
					path := keyDirEntry.filePath
					offset := keyDirEntry.fileEntryOffset
					valSize := keyDirEntry.valueSize
					if path == filePath && offset == curOffset && valSize == valueSize {
						_, err := mergeFilePtr.WriteAt(buf1, b.mergeFileOffset)
						if err != nil {
							fmt.Printf("Error occurred while writing entry: %v\n", err)
							fileMergedSuccessfully = false
							break
						}
						fmt.Printf("Updating the entry for key: %s\n", keyStr)
						pendingUpdates[keyStr] = mergeUpdate{
							originalFilePath: filePath,
							originalOffset:   curOffset,
							originalValSize:  valueSize,
							newEntry: KeyDirEntry{
								filePath:        mergeFilePath,
								fileEntryOffset: b.mergeFileOffset,
								valueSize:       valSize,
							},
						}
						b.mergeFileOffset += blockSize
					}
				}
			}

			curOffset += blockSize
		}

		b.mtx.Lock()

		for keyStr, update := range pendingUpdates {
			current, ok := b.keyDir[keyStr]
			if ok && current.filePath == update.originalFilePath &&
				current.fileEntryOffset == update.originalOffset &&
				current.valueSize == update.originalValSize {
				b.keyDir[keyStr] = update.newEntry
			}
		}

		if !fileMergedSuccessfully {
			fmt.Printf("Skipping cleanup of %s due to merge errors\n", filePath)
			b.mtx.Unlock()
			continue
		}

		filePtr.Close()
		delete(b.filePtrs, filePath)
		os.Remove(filePath)

		if b.mergeFileOffset >= b.maxFileSizeBytes {
			fmt.Printf("Creating new file counter for merge files when offset is %d\n", b.mergeFileOffset)
			b.mergeFileCnt++
			b.mergeFileOffset = 0

			mergeFilePath = fmt.Sprintf("%smerge-%d.data", b.fileBasePath, b.mergeFileCnt)
			mergeFilePtr, err = os.OpenFile(mergeFilePath, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				fmt.Println("Error occurred while creating filePtr:", err)
				b.mtx.Unlock()
				return
			}
			b.filePtrs[mergeFilePath] = mergeFilePtr
		}

		b.mtx.Unlock()
	}

}

func New() *BitCask {
	return &BitCask{
		fileBasePath:     "./",
		fileCnt:          1,
		mergeFileCnt:     1,
		keyDir:           make(map[string]KeyDirEntry),
		currOffset:       0,
		maxFileSizeBytes: 65536,
		filePtrs:         make(map[string]*os.File),
	}
}

func main() {
	bitCask := New()
	// bitCask.Put([]byte("Hello"), []byte("World"))
	// bitCask.Get([]byte("Hello"))
	// // bitCask.Delete([]byte("Hello"))
	// bitCask.Get([]byte("Hello"))

	for i := range 10000 {
		str := fmt.Sprint(i)
		bitCask.Put([]byte(str), []byte(str))
		val, _ := bitCask.Get([]byte(str))
		if string(val) != str {
			fmt.Println("Bug detected")
		}
	}

	bitCask.Merge()
	bitCask.Merge()

}
