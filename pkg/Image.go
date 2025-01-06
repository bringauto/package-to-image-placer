package package_to_image_placer

import (
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"log"
	"os"
)

type ImageCreator struct {
	TargetDisk *disk.Disk
	sourceDisk *disk.Disk
	targetFile string // Maybe not needed, only used in parted
}

func CloneImage(source, target string) (ImageCreator, error) {
	// CloneImage creates a clone of the image
	imageCreator := new(ImageCreator)
	imageCreator.targetFile = target
	err := error(nil)
	imageCreator.sourceDisk, err = diskfs.Open(source)
	if err != nil {
		return *imageCreator, err
	}
	defer imageCreator.sourceDisk.Close()

	sourceImageSize := imageCreator.sourceDisk.Size
	blockSize := imageCreator.sourceDisk.PhysicalBlocksize
	imageCreator.TargetDisk, err = diskfs.Create(target, sourceImageSize, diskfs.SectorSize(blockSize))
	if err != nil {
		return *imageCreator, err
	}
	defer imageCreator.TargetDisk.Close()

	err = imageCreator.copyPartitionTable()
	if err != nil {
		return *imageCreator, err
	}

	//imageCreator.labelFilesystem()
	err = imageCreator.copyPartitionData()
	if err != nil {
		return *imageCreator, err
	}

	return *imageCreator, nil
}

type PartitionInfo struct {
	PartitionNumber int
	PartitionUUID   string
	FilesystemType  string
}

func GetPartitionInfo(disk *disk.Disk) []PartitionInfo {
	table, err := disk.GetPartitionTable()
	if err != nil {
		log.Fatal(err)
	}
	partitions := []PartitionInfo{}
	//log.Printf("All partitions on disk:\n\n")
	for index, p := range table.GetPartitions() {
		partitionNumber := index + 1
		fs, err := disk.GetFilesystem(partitionNumber)
		if err != nil {
			log.Printf("Error getting filesystem on partition %d: %s\n", partitionNumber, err)
		}
		partition := PartitionInfo{
			PartitionNumber: partitionNumber,
			PartitionUUID:   p.UUID(),
			FilesystemType:  TypeToString(fs.Type()),
		}
		//log.Printf("Partition %d: %s\n", partitionNumber, p.UUID())
		//
		//log.Printf("\tFilesystem Type: %s\n\t\t%s", TypeToString(fs.Type()), fs.Label())
		partitions = append(partitions, partition)
	}
	return partitions
}

func TypeToString(t filesystem.Type) string {
	switch t {
	case filesystem.TypeFat32:
		return "FAT32"
	case filesystem.TypeISO9660:
		return "ISO9660"
	case filesystem.TypeSquashfs:
		return "Squashfs"
	case filesystem.TypeExt4:
		return "Ext4"
	default:
		return "Unknown"
	}
}

func (imageCreator ImageCreator) copyPartitionTable() error {
	// TODO Create file and resize it
	//RunCommand("parted "+imageCreator.targetFile+" --script mklabel gpt", "./", false)

	partitions := []*gpt.Partition{}
	sourcePartitionTable, err := imageCreator.sourceDisk.GetPartitionTable()
	if err != nil {
		return err
	}
	for _, p := range sourcePartitionTable.GetPartitions() {
		gptPartition, ok := p.(*gpt.Partition)
		if !ok {
			return fmt.Errorf("failed to assert partition type to gpt.Partition")
		}
		newPartition := &gpt.Partition{
			Start:      gptPartition.Start,
			End:        gptPartition.End, //uint64(start + size/imageCreator.blockSize - 1),
			Size:       gptPartition.Size,
			Type:       gptPartition.Type,
			Name:       gptPartition.Name,
			Attributes: gptPartition.Attributes,
		}
		partitions = append(partitions, newPartition)
	}

	table := &gpt.Table{
		ProtectiveMBR:      true, // TODO Need to get it from source disk
		LogicalSectorSize:  int(imageCreator.sourceDisk.LogicalBlocksize),
		PhysicalSectorSize: int(imageCreator.sourceDisk.PhysicalBlocksize),
		Partitions:         partitions,
	}

	err = imageCreator.TargetDisk.Partition(table)
	if err != nil {
		return err
	}

	return nil
}

func (imageCreator ImageCreator) copyPartitionData() error {
	sourcePartitionTable, err := imageCreator.sourceDisk.GetPartitionTable()
	if err != nil {
		return err
	}
	for index, p := range sourcePartitionTable.GetPartitions() {
		log.Printf("Writing to %d partition: %s", index+1, p.UUID())

		tmpFile, err := os.CreateTemp("", "partition_data_*.tmp")
		if err != nil {
			return err
		}

		bytesRead, err := imageCreator.sourceDisk.ReadPartitionContents(index+1, tmpFile)
		println("bytes read: ", bytesRead)
		if err != nil || bytesRead == 0 {
			return err
		}
		tmpFile.Sync()
		tmpFile.Close()

		reader, err := os.Open(tmpFile.Name()) // Needs to be reopen, doesn't work otherwise (write 0 bytes)
		if err != nil {
			return err
		}
		written, err := imageCreator.TargetDisk.WritePartitionContents(index+1, reader)
		println("written bytes: ", written)
		if err != nil {
			return err
		}
		reader.Close()
		os.Remove(tmpFile.Name())
	}
	return nil
}
