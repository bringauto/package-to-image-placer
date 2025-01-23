package image

import (
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"log"
	"os"
)

type imageCreator struct {
	targetDisk *disk.Disk
	sourceDisk *disk.Disk
}

// CloneImage creates new image and clones source image to it
func CloneImage(source, target string) error {
	log.Printf("Cloning image from %s to %s", source, target)

	imageCreator := new(imageCreator)
	err := error(nil)
	imageCreator.sourceDisk, err = diskfs.Open(source)
	if err != nil {
		return err
	}
	defer imageCreator.sourceDisk.Close()

	sourceImageSize := imageCreator.sourceDisk.Size
	blockSize := imageCreator.sourceDisk.PhysicalBlocksize
	imageCreator.targetDisk, err = diskfs.Create(target, sourceImageSize, diskfs.SectorSize(blockSize))
	if err != nil {
		return err
	}
	defer imageCreator.targetDisk.Close()

	err = imageCreator.copyPartitionTable()
	if err != nil {
		return err
	}

	//imageCreator.labelFilesystem()
	err = imageCreator.copyPartitionData()
	if err != nil {
		return err
	}

	return nil
}

// copyPartitionTable copies partition table from source disk to target disk
// Only GPT partition table is supported. Always sets protective MBR to true
func (imageCreator imageCreator) copyPartitionTable() error {
	var partitions []*gpt.Partition
	sourcePartitionTable, err := imageCreator.sourceDisk.GetPartitionTable()
	if err != nil {
		return err
	}
	if sourcePartitionTable.Type() != "gpt" {
		return fmt.Errorf("source disk partition table is not GPT. Only GPT is supported")
	}
	for _, p := range sourcePartitionTable.GetPartitions() {
		gptPartition, ok := p.(*gpt.Partition)
		if !ok {
			return fmt.Errorf("failed to assert partition type to gpt.Partition")
		}
		newPartition := &gpt.Partition{
			Start:      gptPartition.Start,
			End:        gptPartition.End,
			Size:       gptPartition.Size,
			Type:       gptPartition.Type,
			Name:       gptPartition.Name,
			Attributes: gptPartition.Attributes,
		}
		partitions = append(partitions, newPartition)
	}

	table := &gpt.Table{
		ProtectiveMBR:      true,
		LogicalSectorSize:  int(imageCreator.sourceDisk.LogicalBlocksize),
		PhysicalSectorSize: int(imageCreator.sourceDisk.PhysicalBlocksize),
		Partitions:         partitions,
	}

	err = imageCreator.targetDisk.Partition(table)
	if err != nil {
		return err
	}

	return nil
}

// copyPartitionData copies all data from source disk partitions to target disk partitions
func (imageCreator imageCreator) copyPartitionData() error {
	sourcePartitionTable, err := imageCreator.sourceDisk.GetPartitionTable()
	if err != nil {
		return err
	}
	for index, p := range sourcePartitionTable.GetPartitions() {
		log.Printf("Writing to partition  %d: %s", index+1, p.UUID())

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

		reader, err := os.Open(tmpFile.Name()) // Needs to be reopened, doesn't work otherwise (write 0 bytes)
		if err != nil {
			return err
		}
		written, err := imageCreator.targetDisk.WritePartitionContents(index+1, reader)
		println("written bytes: ", written)
		if err != nil {
			return err
		}
		reader.Close()
		os.Remove(tmpFile.Name())
	}
	return nil
}
