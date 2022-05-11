package main

import (
	"fmt"
	"math"
)

func main() {
	distanceInfo := DistanceInfo{
		deliveryPointDetail: []*DeliveryPointDetail{
			&DeliveryPointDetail{AlgoAddrId: "1", AlgoAddrType: "station", OrderSequence: 0},
			&DeliveryPointDetail{OrderId: "1", AlgoAddrId: "1", AlgoAddrType: "order_point", OrderSequence: 1},
			&DeliveryPointDetail{OrderId: "2", AlgoAddrId: "2", AlgoAddrType: "order_point", OrderSequence: 2},
			&DeliveryPointDetail{OrderId: "3", AlgoAddrId: "2", AlgoAddrType: "order_point", OrderSequence: 3},
			&DeliveryPointDetail{OrderId: "4", AlgoAddrId: "4", AlgoAddrType: "order_point", OrderSequence: 4},
			&DeliveryPointDetail{OrderId: "5", AlgoAddrId: "5", AlgoAddrType: "order_point", OrderSequence: 5}},

		subSequenceDetail: []*SubSequenceDetail{
			&SubSequenceDetail{Distance: 10},
			&SubSequenceDetail{Distance: 10},
			&SubSequenceDetail{Distance: 100},
			&SubSequenceDetail{Distance: 10},
			&SubSequenceDetail{Distance: 10},
		},

		routeDistanceType: "travel",

		consolidateDistance: 50,
	}
	for _, _ = range distanceInfo.deliveryPointDetail {
		distance := distanceInfo.GetDistanceToNextOrder()
		fmt.Println(distance)
	}
}

type DeliveryPointDetail struct {
	// 订单编号;如果algo_addr_type为station则留空
	OrderId string `protobuf:"bytes,1,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
	// 如果algo_addr_type为station,则为station_id,否则为算法计算的poi_id
	AlgoAddrId string `protobuf:"bytes,2,opt,name=algo_addr_id,json=algoAddrId,proto3" json:"algo_addr_id,omitempty"`
	// 二选一文本,'station'/'order_point'
	AlgoAddrType string `protobuf:"bytes,3,opt,name=algo_addr_type,json=algoAddrType,proto3" json:"algo_addr_type,omitempty"`
	// 订单排序,如果poi_type为station,则为0(oneway或roundtrip的起点)或n(round_trip的重点order数量+1)
	OrderSequence int64 `protobuf:"varint,4,opt,name=order_sequence,json=orderSequence,proto3" json:"order_sequence,omitempty"`
	// input地址,reg_addr正则后的用户地址拼接
	Addr string `protobuf:"bytes,5,opt,name=addr,proto3" json:"addr,omitempty"`
	// 算法内部求解地址,即delivery_point对应地址
	AlgoAddr string `protobuf:"bytes,6,opt,name=algo_addr,json=algoAddr,proto3" json:"algo_addr,omitempty"`
	// 地址经纬度
	AddrLat float64 `protobuf:"fixed64,7,opt,name=addr_lat,json=addrLat,proto3" json:"addr_lat,omitempty"`
	AddrLng float64 `protobuf:"fixed64,8,opt,name=addr_lng,json=addrLng,proto3" json:"addr_lng,omitempty"`
	// 该order的重量, 单位g, 若algo_addr_type为station则为0
	OrderWeight int64 `protobuf:"varint,9,opt,name=order_weight,json=orderWeight,proto3" json:"order_weight,omitempty"`
	// 预留字段,预计到该点时间
	StartTime int64 `protobuf:"varint,10,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	// 预留字段,预计离开该点时间
	EndTime int64 `protobuf:"varint,11,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
}

type DistanceInfo struct {
	deliveryPointDetail []*DeliveryPointDetail
	subSequenceDetail   []*SubSequenceDetail

	currentPoiId            string
	currentPoiDistanceIndex int64
	poiDistanceIndex        int64

	routeDistanceType string

	consolidateDistance float64

	orderIndex int

	distance float64
}

type SubSequenceDetail struct {
	// 起始algo计算点（某个algo_addr_id）
	StartAlgoAddrId string `protobuf:"bytes,1,opt,name=start_algo_addr_id,json=startAlgoAddrId,proto3" json:"start_algo_addr_id,omitempty"`
	// 结束algo计算点（路线下一个algo_addr_id)
	EndAlgoAddrId string `protobuf:"bytes,2,opt,name=end_algo_addr_id,json=endAlgoAddrId,proto3" json:"end_algo_addr_id,omitempty"`
	// 起始点和结束点的距离, 单位m
	Distance int64 `protobuf:"varint,3,opt,name=distance,proto3" json:"distance,omitempty"`
	// 起始poi到结束poi的成本, 单位美分
	TravelCost int64 `protobuf:"varint,4,opt,name=travel_cost,json=travelCost,proto3" json:"travel_cost,omitempty"`
	// 起始poi到结束poi的行驶距离, 单位m
	TravelTime int64 `protobuf:"varint,5,opt,name=travel_time,json=travelTime,proto3" json:"travel_time,omitempty"`
}

// GetDistanceToNextOrder 获取本次订单同下一单之间的距离
func (di *DistanceInfo) GetDistanceToNextOrder() float64 {
	fmt.Println("orderIndex:", di.orderIndex)
	defer func() {
		di.orderIndex++
	}()
	if di.orderIndex >= len(di.deliveryPointDetail)-1 {
		return 0
	}
	if di.deliveryPointDetail[di.orderIndex].AlgoAddrType == "station" {
		di.currentPoiId = ""
		di.poiDistanceIndex++
		//di.currentPoiDistanceIndex = di.poiDistanceIndex // subSequenceDetail 中记录的是当前poi到下一个poi点之间的距离，这里应该跳过station
		return 0
	}
	if di.deliveryPointDetail[di.orderIndex+1].AlgoAddrType == "station" {
		di.currentPoiId = ""
		di.poiDistanceIndex++
		//di.currentPoiDistanceIndex = di.poiDistanceIndex
		return math.MaxInt64 // 订单跟站点之间的距离默认为无穷大，也即是不将站点聚合到订单中。
	}

	if di.currentPoiId == "" {
		di.currentPoiId = di.deliveryPointDetail[di.orderIndex].AlgoAddrId
	}
	fmt.Println(di.currentPoiId, di.poiDistanceIndex, di.currentPoiDistanceIndex)
	if di.routeDistanceType == "straight" {
		// 直线距离：聚合的范围比较小，默认是50米，距离使用球面距离。
		/*
			return geometry.CalculateGreatCircle(di.deliveryPointDetail[di.orderIndex].AddrLat,
				di.deliveryPointDetail[di.orderIndex].AddrLng,
				di.deliveryPointDetail[di.orderIndex+1].AddrLat,
				di.deliveryPointDetail[di.orderIndex+1].AddrLng)


		*/
		return 0
	} else if di.routeDistanceType == "travel" {
		// 交通距离：根据配置的车辆计算交通距离。
		if di.deliveryPointDetail[di.orderIndex].AlgoAddrId == di.deliveryPointDetail[di.orderIndex+1].AlgoAddrId {
			return di.distance
		} else {
			di.distance += float64(di.subSequenceDetail[di.poiDistanceIndex].Distance)
			di.poiDistanceIndex++
			ret := di.distance
			if di.distance > di.consolidateDistance {
				di.distance = 0
				di.currentPoiId = di.deliveryPointDetail[di.orderIndex+1].AlgoAddrId
				//di.currentPoiDistanceIndex = di.poiDistanceIndex
			}
			return ret
		}
		/*
			// 交通距离：根据配置的车辆计算交通距离。
			if di.deliveryPointDetail[di.orderIndex+1].AlgoAddrId == di.currentPoiId {
				return 0 // 算法对订单根据poi进行了聚合，同一个poi的订单距离默认为0
			} else {
				if di.deliveryPointDetail[di.orderIndex].AlgoAddrId != di.deliveryPointDetail[di.orderIndex+1].AlgoAddrId {
					di.poiDistanceIndex++
				}
				var distance float64
				fmt.Println("distance", distance, di.currentPoiDistanceIndex, di.poiDistanceIndex)
				for i := di.currentPoiDistanceIndex; i < di.poiDistanceIndex; i++ {
					distance += float64(di.subSequenceDetail[i].Distance)
				}
				if distance > di.consolidateDistance {
					di.currentPoiId = di.deliveryPointDetail[di.orderIndex+1].AlgoAddrId
					di.currentPoiDistanceIndex = di.poiDistanceIndex
				}
				return distance
			}

		*/
	}
	return 0
}
