package db

import "gorm.io/gorm"

const (
	columnNumber = "Number"
	columnSumNumber = "Sum_Number"
)

func CreateVoteDetail(voteDetail *VoteDetail) error {
	// 此处是创建，而非更新
	err := GSqlInstance.Create(&voteDetail).Error
	return err
}

func UpdateVoteDetailNumber(voteId int64, id int64) error{
	return GSqlInstance.Model(&VoteDetail{}).Where("id = ? and voteId = ?", id, voteId).
		Update(columnNumber, gorm.Expr("Number + ?", 1)).
		Update(columnSumNumber,gorm.Expr("Sum_Number + ?", 1)).Error

}

func GetVoteDetail(offset int, limit int) (int, []*VoteDetail, error) {
	var (
		count     int64
		groupList []*VoteDetail
		err       error
	)

	if err = GSqlInstance.Model(&VoteDetail{}).Count(&count).Error; err != nil { //计算得到一个模型的多少条记录
		return int(count), groupList, err
	}

	if err = GSqlInstance.Model(&VoteDetail{}).
		Offset(offset).Limit(limit).Find(&groupList).Error; err != nil {
		return int(count), groupList, err
	}
	return int(count), groupList, err
}
