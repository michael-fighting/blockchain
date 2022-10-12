package db

func CreateVote(vote *Vote) error {
	// 此处是创建，而非更新
	err := GSqlInstance.Create(&vote).Error
	return err
}


func GetVoteList(offset int, limit int) (int, []*Vote, error) {
	var (
		count     int64
		groupList []*Vote
		err       error
	)

	if err = GSqlInstance.Model(&Vote{}).Count(&count).Error; err != nil { //计算得到一个模型的多少条记录
		return int(count), groupList, err
	}

	if err = GSqlInstance.Model(&Vote{}).
		Offset(offset).Limit(limit).Find(&groupList).Error; err != nil {
		return int(count), groupList, err
	}
	return int(count), groupList, err
}


func GetVoteByUserId(voteId int) (*Vote, error) {
	var vote Vote
	if err := GSqlInstance.Where("Id = ?", voteId).Find(&vote).Error; err != nil {
		return nil, err
	}
	return &vote, nil
}

//func GetBookmarkByUserId(userId string) (*common.Bookmark, error) {
//	var bookmark common.Bookmark
//	if err := connection.DB.Where("bookmark_userId = ?", userId).Find(&bookmark).Error; err != nil {
//		log.Error("GetUserByBookmarkName Failed: " + err.Error())
//		return nil, err
//	}
//	return &bookmark, nil
//}
