package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type KeyDirEntry struct {
	filePath        string
	fileEntryOffset int64
	valueSize       uint32
}

type BitCask struct {
	fileBasePath        string
	currOffset          int64
	filePtrs            map[string]*os.File
	keyDir              map[string]KeyDirEntry
	mtx                 sync.RWMutex
	fileCnt             int
	maxFileSizeBytes    int64
	mergeFileCnt        int
	mergeFileOffset     int64
	mergeHintFileOffset int64
}

type mergeUpdate struct {
	originalFilePath string
	originalOffset   int64
	originalValSize  uint32
	newEntry         KeyDirEntry
}

type mergeSegment struct {
	id         int
	dataPath   string
	hintPath   string
	dataFile   *os.File
	hintFile   *os.File
	dataOffset int64
	hintOffset int64
}

const headerSize = 17 // crc(4) + tstamp(4) + ksz(4) + vsz(4) + flag(1)

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
	headerBuf := make([]byte, headerSize)
	_, err := filePtr.ReadAt(headerBuf, keyDirEntry.fileEntryOffset)
	if err != nil {
		fmt.Println("Error occurred while reading from offset:", err)
		b.mtx.RUnlock()
		return nil, err
	}

	storedKeySize := binary.LittleEndian.Uint32(headerBuf[8:12])
	storedValueSize := binary.LittleEndian.Uint32(headerBuf[12:16])
	flag := headerBuf[16]

	buf := make([]byte, headerSize+int(storedKeySize)+int(storedValueSize))
	_, err = filePtr.ReadAt(buf, keyDirEntry.fileEntryOffset)
	b.mtx.RUnlock()
	if err != nil {
		fmt.Println("Error occurred while reading full entry from offset:", err)
		return nil, err
	}

	crc := binary.LittleEndian.Uint32(buf[0:4])
	checkSum := crc32.ChecksumIEEE(buf[4:])
	if checkSum != crc {
		fmt.Println("Checksum doesn't match for key")
		return nil, errors.New("Corrupted entry found")
	}

	storedKey := buf[headerSize : headerSize+int(storedKeySize)]
	if string(storedKey) != string(key) {
		fmt.Println("Stored key doesn't match requested key")
		return nil, errors.New("Key mismatch")
	}

	if flag == 1 {
		fmt.Println("Key is marked deleted")
		return nil, errors.New("Key not found")
	}

	value := buf[headerSize+int(storedKeySize):]
	fmt.Println("Returned value: ", string(value))
	return value, nil
}

func (b *BitCask) Merge() {
	b.mtx.Lock()
	activeFilePath := fmt.Sprintf("%s%d.data", b.fileBasePath, b.fileCnt)
	segment, err := b.openMergeSegment(b.mergeFileCnt)
	if err != nil {
		fmt.Println("Error occurred while creating merge segment:", err)
		b.mtx.Unlock()
		return
	}
	b.filePtrs[segment.dataPath] = segment.dataFile

	// Snapshot all immutable file paths (everything except active and current merge file)
	var immutableFiles []string
	for path := range b.filePtrs {
		if path != activeFilePath && path != segment.dataPath {
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
						b.mtx.Lock()
						segment, err = b.ensureMergeSegmentCapacityLocked(segment, blockSize)
						if err != nil {
							fmt.Printf("Error occurred while rotating merge segment: %v\n", err)
							b.mtx.Unlock()
							fileMergedSuccessfully = false
							break
						}

						newEntry, err := segment.appendEntry(buf1, key, valueSize)
						b.mergeFileOffset = segment.dataOffset
						b.mergeHintFileOffset = segment.hintOffset
						b.mtx.Unlock()
						if err != nil {
							fmt.Printf("Error occurred while appending merged entry: %v\n", err)
							fileMergedSuccessfully = false
							break
						}

						fmt.Printf("Updating the entry for key: %s\n", keyStr)
						pendingUpdates[keyStr] = mergeUpdate{
							originalFilePath: filePath,
							originalOffset:   curOffset,
							originalValSize:  valueSize,
							newEntry:         newEntry,
						}
					}
				}
			}

			curOffset += blockSize
		}

		b.mtx.Lock()

		// Sync the merge files to disk before you do deletion of the
		err = segment.sync()
		if err != nil {
			fmt.Printf("Error occurred while syncing merge segment: %v\n", err)
			b.mtx.Unlock()
			continue
		}

		if !fileMergedSuccessfully {
			fmt.Printf("Skipping cleanup of %s due to merge errors\n", filePath)
			b.mtx.Unlock()
			continue
		}

		for keyStr, update := range pendingUpdates {
			current, ok := b.keyDir[keyStr]
			if ok && current.filePath == update.originalFilePath &&
				current.fileEntryOffset == update.originalOffset &&
				current.valueSize == update.originalValSize {
				b.keyDir[keyStr] = update.newEntry
			}
		}

		filePtr.Close()
		delete(b.filePtrs, filePath)
		os.Remove(filePath)

		if strings.Contains(filePath, "merge") {
			hintFilePath := strings.TrimSuffix(filePath, ".data") + ".hint"
			os.Remove(hintFilePath)
		}

		b.mergeFileOffset = segment.dataOffset
		b.mergeHintFileOffset = segment.hintOffset
		b.mtx.Unlock()
	}

	err = segment.closeHint()
	if err != nil {
		fmt.Println("Error occurred while closing merge hint file:", err)
	}

}

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

func (b *BitCask) openMergeSegment(id int) (*mergeSegment, error) {
	dataPath := fmt.Sprintf("%smerge-%d.data", b.fileBasePath, id)
	hintPath := fmt.Sprintf("%smerge-%d.hint", b.fileBasePath, id)

	dataAlreadyExists := true
	_, err := os.Stat(dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			dataAlreadyExists = false
		} else {
			return nil, err
		}
	}

	dataFile, err := os.OpenFile(dataPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	hintFile, err := os.OpenFile(hintPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		dataFile.Close()
		if !dataAlreadyExists {
			os.Remove(dataPath)
		}
		return nil, err
	}

	dataInfo, err := dataFile.Stat()
	if err != nil {
		dataFile.Close()
		hintFile.Close()
		return nil, err
	}

	hintInfo, err := hintFile.Stat()
	if err != nil {
		dataFile.Close()
		hintFile.Close()
		return nil, err
	}

	return &mergeSegment{
		id:         id,
		dataPath:   dataPath,
		hintPath:   hintPath,
		dataFile:   dataFile,
		hintFile:   hintFile,
		dataOffset: dataInfo.Size(),
		hintOffset: hintInfo.Size(),
	}, nil
}

func (b *BitCask) rotateMergeSegmentLocked(segment *mergeSegment) (*mergeSegment, error) {
	if err := segment.sync(); err != nil {
		return nil, err
	}
	if err := segment.closeHint(); err != nil {
		return nil, err
	}

	b.mergeFileCnt++
	nextSegment, err := b.openMergeSegment(b.mergeFileCnt)
	if err != nil {
		return nil, err
	}
	b.filePtrs[nextSegment.dataPath] = nextSegment.dataFile
	return nextSegment, nil
}

func (b *BitCask) ensureMergeSegmentCapacityLocked(segment *mergeSegment, nextDataSize int64) (*mergeSegment, error) {
	if segment.dataOffset+nextDataSize <= b.maxFileSizeBytes {
		return segment, nil
	}

	fmt.Printf("Creating new file counter for merge files when offset is %d\n", segment.dataOffset)
	return b.rotateMergeSegmentLocked(segment)
}

func (m *mergeSegment) sync() error {
	if err := m.dataFile.Sync(); err != nil {
		return err
	}
	if err := m.hintFile.Sync(); err != nil {
		return err
	}
	return nil
}

func (m *mergeSegment) closeHint() error {
	return m.hintFile.Close()
}

func (m *mergeSegment) appendEntry(entryBuf []byte, key []byte, valueSize uint32) (KeyDirEntry, error) {
	_, err := m.dataFile.WriteAt(entryBuf, m.dataOffset)
	if err != nil {
		return KeyDirEntry{}, err
	}

	hintRecordBuf := make([]byte, 16+len(key))
	binary.LittleEndian.PutUint32(hintRecordBuf[0:4], uint32(len(key)))
	binary.LittleEndian.PutUint32(hintRecordBuf[4:8], valueSize)
	binary.LittleEndian.PutUint64(hintRecordBuf[8:16], uint64(m.dataOffset))
	copy(hintRecordBuf[16:], key)

	_, err = m.hintFile.WriteAt(hintRecordBuf, m.hintOffset)
	if err != nil {
		return KeyDirEntry{}, err
	}

	entry := KeyDirEntry{
		filePath:        m.dataPath,
		fileEntryOffset: m.dataOffset,
		valueSize:       valueSize,
	}

	m.dataOffset += int64(len(entryBuf))
	m.hintOffset += int64(len(hintRecordBuf))

	return entry, nil
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
		bitCask.Get([]byte(str))
	}

	// bitCask.Merge()
	// bitCask.Merge()

}
