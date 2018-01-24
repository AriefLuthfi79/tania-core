package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/Tanibox/tania-server/src/assets/domain"
	"github.com/Tanibox/tania-server/src/assets/query"
	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
)

type SimpleFarm domain.Farm
type SimpleArea struct {
	UID  uuid.UUID
	Name string
	Type string
}
type AreaList struct {
	UID            uuid.UUID `json:"uid"`
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	TotalCropBatch int       `json:"total_crop_batch"`
	PlantQuantity  int       `json:"plant_quantity"`
}
type DetailArea struct {
	UID            uuid.UUID                   `json:"uid"`
	Name           string                      `json:"name"`
	Size           domain.AreaSize             `json:"size"`
	Type           string                      `json:"type"`
	Location       string                      `json:"location"`
	Photo          domain.AreaPhoto            `json:"photo"`
	Reservoir      domain.Reservoir            `json:"reservoir"`
	TotalCropBatch int                         `json:"total_crop_batch"`
	TotalVariety   int                         `json:"total_variety"`
	Crops          []query.AreaCropQueryResult `json:"crops"`
	Notes          SortedAreaNotes             `json:"notes"`
}

type DetailReservoir struct {
	domain.Reservoir
	InstalledToAreas []SimpleArea
}

type ReservoirBucket struct{ domain.Bucket }
type ReservoirTap struct{ domain.Tap }
type PlantType struct{ domain.PlantType }

type InventoryMaterial struct {
	UID       uuid.UUID `json:"uid"`
	PlantType PlantType `json:"plant_type"`
	Variety   string    `json:"variety"`
}

type AvailableInventory struct {
	PlantType PlantType `json:"plant_type"`
	Varieties []string  `json:"varieties"`
}

type SortedAreaNotes []domain.AreaNote

// Len is part of sort.Interface.
func (sn SortedAreaNotes) Len() int { return len(sn) }

// Swap is part of sort.Interface.
func (sn SortedAreaNotes) Swap(i, j int) { sn[i], sn[j] = sn[j], sn[i] }

// Less is part of sort.Interface.
func (sn SortedAreaNotes) Less(i, j int) bool { return sn[i].CreatedDate.After(sn[j].CreatedDate) }

type SortedReservoirNotes []domain.ReservoirNote

// Len is part of sort.Interface.
func (sn SortedReservoirNotes) Len() int { return len(sn) }

// Swap is part of sort.Interface.
func (sn SortedReservoirNotes) Swap(i, j int) { sn[i], sn[j] = sn[j], sn[i] }

// Less is part of sort.Interface.
func (sn SortedReservoirNotes) Less(i, j int) bool { return sn[i].CreatedDate.After(sn[j].CreatedDate) }

func MapToSimpleFarm(farms []domain.Farm) []SimpleFarm {
	farmList := make([]SimpleFarm, len(farms))

	for i, farm := range farms {
		farmList[i] = SimpleFarm(farm)
	}

	return farmList
}

func MapToSimpleArea(areas []domain.Area) []SimpleArea {
	simpleAreaList := make([]SimpleArea, len(areas))

	for i, area := range areas {
		simpleAreaList[i] = SimpleArea{
			UID:  area.UID,
			Name: area.Name,
			Type: area.Type.Code,
		}
	}

	return simpleAreaList
}

func MapToAreaList(s *FarmServer, areas []domain.Area) ([]AreaList, error) {
	areaList := make([]AreaList, len(areas))

	for i, area := range areas {
		queryResult := <-s.CropQuery.CountCropsByArea(area.UID)
		if queryResult.Error != nil {
			return []AreaList{}, queryResult.Error
		}

		cropCount, ok := queryResult.Result.(query.CountAreaCropQueryResult)
		if !ok {
			return []AreaList{}, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
		}

		areaList[i] = AreaList{
			UID:            area.UID,
			Name:           area.Name,
			Type:           area.Type.Code,
			TotalCropBatch: cropCount.TotalCropBatch,
			PlantQuantity:  cropCount.PlantQuantity,
		}
	}

	return areaList, nil
}

func MapToReservoir(s *FarmServer, reservoirs []domain.Reservoir) ([]DetailReservoir, error) {
	reservoirList := make([]DetailReservoir, len(reservoirs))

	for i, reservoir := range reservoirs {
		reservoirList[i] = DetailReservoir{Reservoir: reservoir}

		switch v := reservoir.WaterSource.(type) {
		case domain.Bucket:
			reservoirList[i].WaterSource = ReservoirBucket{Bucket: v}
		case domain.Tap:
			reservoirList[i].WaterSource = ReservoirTap{Tap: v}
		}

		queryResult := <-s.AreaQuery.FindAreasByReservoirID(reservoir.UID.String())
		if queryResult.Error != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Internal server error")
		}

		areas, ok := queryResult.Result.([]domain.Area)
		if !ok {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "Internal server error")
		}

		reservoirList[i].InstalledToAreas = MapToSimpleArea(areas)
	}

	return reservoirList, nil
}

func MapToDetailReservoir(s *FarmServer, reservoir domain.Reservoir) (DetailReservoir, error) {
	detailReservoir := DetailReservoir{Reservoir: reservoir}

	switch v := detailReservoir.WaterSource.(type) {
	case domain.Bucket:
		detailReservoir.WaterSource = ReservoirBucket{Bucket: v}
	case domain.Tap:
		detailReservoir.WaterSource = ReservoirTap{Tap: v}
	}

	queryResult := <-s.AreaQuery.FindAreasByReservoirID(detailReservoir.UID.String())
	if queryResult.Error != nil {
		return DetailReservoir{}, echo.NewHTTPError(http.StatusBadRequest, "Internal server error")
	}

	areas, ok := queryResult.Result.([]domain.Area)
	if !ok {
		return DetailReservoir{}, echo.NewHTTPError(http.StatusBadRequest, "Internal server error")
	}

	detailReservoir.InstalledToAreas = MapToSimpleArea(areas)

	return detailReservoir, nil
}

func MapToDetailArea(s *FarmServer, area domain.Area) (DetailArea, error) {
	detailArea := DetailArea{}

	detailArea.UID = area.UID
	detailArea.Name = area.Name
	detailArea.Type = area.Type.Code
	detailArea.Location = area.Location.Code
	detailArea.Photo = area.Photo
	detailArea.Size = area.Size

	detailArea.Reservoir = area.Reservoir
	switch v := area.Reservoir.WaterSource.(type) {
	case domain.Bucket:
		detailArea.Reservoir.WaterSource = ReservoirBucket{Bucket: v}
	case domain.Tap:
		detailArea.Reservoir.WaterSource = ReservoirTap{Tap: v}
	}

	queryResult := <-s.CropQuery.CountCropsByArea(area.UID)
	if queryResult.Error != nil {
		return DetailArea{}, queryResult.Error
	}

	cropCount, ok := queryResult.Result.(query.CountAreaCropQueryResult)
	if !ok {
		return DetailArea{}, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
	}

	detailArea.TotalCropBatch = cropCount.TotalCropBatch

	queryResult = <-s.CropQuery.FindAllCropByArea(area.UID)
	if queryResult.Error != nil {
		return DetailArea{}, queryResult.Error
	}

	crops, ok := queryResult.Result.([]query.AreaCropQueryResult)
	if !ok {
		return DetailArea{}, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
	}

	for i, v := range crops {
		repoResult := <-s.AreaRepo.FindByID(v.InitialArea.AreaUID.String())

		if repoResult.Error != nil {
			return DetailArea{}, queryResult.Error
		}

		a, ok := repoResult.Result.(domain.Area)
		if !ok {
			return DetailArea{}, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
		}

		repoResult = <-s.InventoryMaterialRepo.FindByID(v.Inventory.UID.String())
		if repoResult.Error != nil {
			return DetailArea{}, repoResult.Error
		}

		inv, ok := repoResult.Result.(domain.InventoryMaterial)
		if !ok {
			return DetailArea{}, echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
		}

		now := time.Now()
		diff := now.Sub(v.CreatedDate)
		days := int(diff.Hours()) / 24

		crops[i].DaysSinceSeeding = days
		crops[i].InitialArea.Name = a.Name
		crops[i].Inventory.PlantType = inv.PlantType.Code()
		crops[i].Inventory.Variety = inv.Variety
	}

	detailArea.Crops = crops

	uniqueInventories := make(map[uuid.UUID]bool)
	for _, v := range crops {
		if _, ok := uniqueInventories[v.Inventory.UID]; !ok {
			uniqueInventories[v.Inventory.UID] = true
		}
	}

	detailArea.TotalVariety = len(uniqueInventories)

	notes := make(SortedAreaNotes, 0, len(area.Notes))
	for _, v := range area.Notes {
		notes = append(notes, v)
	}

	sort.Sort(notes)

	detailArea.Notes = notes

	return detailArea, nil
}

func MapToPlantType(plantTypes []domain.PlantType) []PlantType {
	pt := make([]PlantType, len(plantTypes))

	for i, v := range plantTypes {
		pt[i] = PlantType{PlantType: v}
	}

	return pt
}

func MapToAvailableInventories(inventories []domain.InventoryMaterial) []AvailableInventory {
	ai := make(map[string]AvailableInventory, 0)

	// Convert domain.InventoryMaterial to AvailableInventory first with Map
	for _, v := range inventories {
		inv := AvailableInventory{
			PlantType: PlantType{PlantType: v.PlantType},
			Varieties: append(ai[v.PlantType.Code()].Varieties, v.Variety),
		}

		ai[v.PlantType.Code()] = inv
	}

	// From Map, we need to change it to slice for the json response purpose
	aiSlice := []AvailableInventory{}
	for _, v := range ai {
		aiSlice = append(aiSlice, v)
	}

	return aiSlice
}

func MapToInventoryMaterial(inventoryMaterial domain.InventoryMaterial) InventoryMaterial {
	return InventoryMaterial{
		UID:       inventoryMaterial.UID,
		PlantType: PlantType{PlantType: inventoryMaterial.PlantType},
		Variety:   inventoryMaterial.Variety,
	}
}

func (sf SimpleFarm) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		UID  string `json:"uid"`
		Name string `json:"name"`
		Type string `json:"type"`
	}{
		UID:  sf.UID.String(),
		Name: sf.Name,
		Type: sf.Type,
	})
}

func (sa SimpleArea) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		UID  string `json:"uid"`
		Name string `json:"name"`
		Type string `json:"type"`
	}{
		UID:  sa.UID.String(),
		Name: sa.Name,
		Type: sa.Type,
	})
}

func (dr DetailReservoir) MarshalJSON() ([]byte, error) {
	notes := make(SortedReservoirNotes, 0, len(dr.Notes))
	for _, v := range dr.Notes {
		notes = append(notes, v)
	}

	sort.Sort(notes)

	return json.Marshal(struct {
		UID              string               `json:"uid"`
		Name             string               `json:"name"`
		PH               float32              `json:"ph"`
		EC               float32              `json:"ec"`
		Temperature      float32              `json:"temperature"`
		WaterSource      domain.WaterSource   `json:"water_source"`
		Notes            SortedReservoirNotes `json:"notes"`
		CreatedDate      time.Time            `json:"created_date"`
		InstalledToAreas []SimpleArea         `json:"installed_to_areas"`
	}{
		UID:              dr.UID.String(),
		Name:             dr.Name,
		PH:               dr.PH,
		EC:               dr.EC,
		Temperature:      dr.Temperature,
		WaterSource:      dr.WaterSource,
		Notes:            notes,
		CreatedDate:      dr.CreatedDate,
		InstalledToAreas: dr.InstalledToAreas,
	})
}

func (rb ReservoirBucket) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type     string  `json:"type"`
		Capacity float32 `json:"capacity"`
		Volume   float32 `json:"volume"`
	}{
		Type:     rb.Type(),
		Capacity: rb.Capacity,
		Volume:   rb.Volume,
	})
}

func (rt ReservoirTap) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
	}{
		Type: rt.Type(),
	})
}

func (pt PlantType) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Code string `json:"code"`
	}{
		Code: pt.PlantType.Code(),
	})
}
