package inmemory

import (
	assetsdomain "github.com/Tanibox/tania-server/src/assets/domain"
	"github.com/Tanibox/tania-server/src/assets/storage"
	"github.com/Tanibox/tania-server/src/growth/query"
	uuid "github.com/satori/go.uuid"
)

type MaterialQueryInMemory struct {
	Storage *storage.MaterialStorage
}

func NewMaterialQueryInMemory(s *storage.MaterialStorage) query.MaterialQuery {
	return MaterialQueryInMemory{Storage: s}
}

func (s MaterialQueryInMemory) FindByID(inventoryUID uuid.UUID) <-chan query.QueryResult {
	result := make(chan query.QueryResult)

	go func() {
		s.Storage.Lock.RLock()
		defer s.Storage.Lock.RUnlock()

		ci := query.CropMaterialQueryResult{}
		for _, val := range s.Storage.MaterialMap {
			if val.UID == inventoryUID {
				ci.UID = val.UID
				ci.Name = val.Name

				// WARNING, domain leakage
				switch v := val.Type.(type) {
				case assetsdomain.MaterialTypeSeed:
					ci.MaterialSeedPlantTypeCode = v.PlantType.Code
				case assetsdomain.MaterialTypePlant:
					ci.MaterialSeedPlantTypeCode = v.PlantType.Code
				}
			}
		}

		result <- query.QueryResult{Result: ci}

		close(result)
	}()

	return result
}

func (q MaterialQueryInMemory) FindMaterialByPlantTypeCodeAndName(plantTypeCode string, name string) <-chan query.QueryResult {
	result := make(chan query.QueryResult)

	go func() {
		q.Storage.Lock.RLock()
		defer q.Storage.Lock.RUnlock()

		ci := query.CropMaterialQueryResult{}
		for _, val := range q.Storage.MaterialMap {
			// WARNING, domain leakage
			switch v := val.Type.(type) {
			case assetsdomain.MaterialTypeSeed:
				if v.PlantType.Code == plantTypeCode && val.Name == name {
					ci.UID = val.UID
					ci.Name = val.Name
					ci.MaterialSeedPlantTypeCode = v.PlantType.Code
				}
			case assetsdomain.MaterialTypePlant:
				if v.PlantType.Code == plantTypeCode && val.Name == name {
					ci.UID = val.UID
					ci.Name = val.Name
					ci.MaterialSeedPlantTypeCode = v.PlantType.Code
				}
			}
		}

		result <- query.QueryResult{Result: ci}

		close(result)
	}()

	return result
}
