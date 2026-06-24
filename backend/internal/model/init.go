package model

import "gkk/tool/migrate"

func Init() {
	migrate.Register(
		new(User),
		new(Merchant),
		new(Product),
		new(StallSession),
		new(Preorder),
		new(Favorite),
		new(Application),
		new(Feedback),
	)
}
