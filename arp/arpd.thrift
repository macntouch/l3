namespace go arpd
typedef i32 int

struct ArpEntry {
        1: string IpAddr,
        2: string MacAddr,
        3: int    Vlan,
        4: string Intf,
        5: string ExpiryTimeLeft,
}
struct ArpEntryBulk {
        1: int StartIdx,
        2: int EndIdx,
        3: int Count,
        4: bool More,
        5: list<ArpEntry> ArpList,
}


service ARPService
{
    int ResolveArpIPV4(1:string destNetIp,2:int iftype, 3:int vlanid);
    int SetArpConfig(1:int Timeout);
    int UpdateUntaggedPortToVlanMap(1:int vlanid, 2:string untaggedMembers);
    ArpEntryBulk GetBulkArpEntry(1:int currMarker, 2:int count)
}
