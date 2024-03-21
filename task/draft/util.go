package draft

import "strconv"

func topic(id uint64) string {
	return "topic_prefix" + strconv.FormatUint(id, 10)
}
