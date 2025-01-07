package models

type ModelEntry struct {
	ID    string `bson:"_id" json:"id"`
	Name  string `bson:"name" json:"name"`
	USDZ  string `bson:"usdz" json:"usdz"`
	GLB   string `bson:"glb" json:"glb"`
}
