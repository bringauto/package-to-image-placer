package package_to_image_placer

import (
	"fmt"
	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"log"
	"os"
	"package-to-image-placer/pkg/interaction"
)

type ImageCreator struct {
	TargetDisk *disk.Disk
	sourceDisk *disk.Disk
	targetFile string // Maybe not needed, only used in parted
}

// CloneImage creates new image and clones source image to it
func CloneImage(source, target string) error {
	log.Printf("Cloning image from %s to %s", source, target)
	if DoesFileExists(target) {
		askUser := fmt.Sprintf("File %s already exists. Do you want to delete it?", target)
		if !interaction.GetUserConfirmation(askUser) {
			return fmt.Errorf("file already exists and user chose not to delete it")
		}
		if err := os.Remove(target); err != nil {
			return fmt.Errorf("unable to delete existing file: %s", err)
		}
	}

	imageCreator := new(ImageCreator)
	imageCreator.targetFile = target
	err := error(nil)
	imageCreator.sourceDisk, err = diskfs.Open(source)
	if err != nil {
		return err
	}
	defer imageCreator.sourceDisk.Close()

	sourceImageSize := imageCreator.sourceDisk.Size
	blockSize := imageCreator.sourceDisk.PhysicalBlocksize
	imageCreator.TargetDisk, err = diskfs.Create(target, sourceImageSize, diskfs.SectorSize(blockSize))
	if err != nil {
		return err
	}
	defer imageCreator.TargetDisk.Close()

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

func (imageCreator ImageCreator) copyPartitionTable() error {
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

		reader, err := os.Open(tmpFile.Name()) // Needs to be reopened, doesn't work otherwise (write 0 bytes)
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
