namespace go ribd
typedef i32 int
struct NextHopInfo {
	1: int NextHopIfType,
    2: string NextHopIp,
    3: int NextHopIfIndex,
	4: int Metric,
}
struct Routes {
	1: string Ipaddr,
	2: string Mask,
	3: string NextHopIp,
	4: int NextHopIfType
	5: int IfIndex,
	6: int Metric,
	7: int Prototype,
	8: bool IsValid,
	9: int SliceIdx,
}
struct RoutesGetInfo {
	1: int StartIdx,
	2: int EndIdx,
	3: int Count,
	4: bool More,
	5: list<Routes> RouteList,
}
//typedef RouteList  list<Routes>
service RouteService 
{
    int createV4Route (1:string destNetIp, 2:string networkMask, 3:int metric, 4:string nextHopIp, 5: int nextHopIfType, 6:int nextHopIfIndex, 7:int routeType);
    void updateV4Route (1:string destNetIp, 2:string networkMask, 3:int routeType, 4:string nextHopIp, 5:int nextHopIfIndex, 6:int metric);
    int deleteV4Route (1:string destNetIp, 2:string networkMask, 3:int routeType);
    NextHopInfo getRouteReachabilityInfo(1: string desIPv4MasktNet);
	list<Routes> getConnectedRoutesInfo();
    void printV4Routes();
	RoutesGetInfo getBulkRoutes(1: int fromIndex, 2: int count);
	Routes getRoute(1: string destNetIp, 2:string networkMask);
	void linkDown(1: int ifType, 2:int ifIndex);
	void linkUp(1: int ifType, 2:int ifIndex);
}
