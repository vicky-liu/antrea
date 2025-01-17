package openflow

import (
	"fmt"
	"net"

	"github.com/contiv/libOpenflow/openflow13"
	"github.com/contiv/ofnet/ofctrl"
)

type ofFlowAction struct {
	builder *ofFlowBuilder
}

// Drop is an action to drop packets.
func (a *ofFlowAction) Drop() FlowBuilder {
	dropAction := a.builder.Flow.Table.Switch.DropAction()
	a.builder.ofFlow.lastAction = dropAction
	return a.builder
}

// Output is an action to output packets to the specified ofport.
func (a *ofFlowAction) Output(port int) FlowBuilder {
	outputAction := ofctrl.NewOutputPort(uint32(port))
	a.builder.ofFlow.lastAction = outputAction
	return a.builder
}

// OutputFieldRange is an action to output packets to the port located in the specified NXM field with rng.
func (a *ofFlowAction) OutputFieldRange(name string, rng Range) FlowBuilder {
	outputAction, _ := ofctrl.NewNXOutput(name, int(rng[0]), int(rng[1]))
	a.builder.ofFlow.lastAction = outputAction
	return a.builder
}

// OutputFieldRange is an action to output packets to a port which is located in the specified NXM register[rng[0]..rng[1]].
func (a *ofFlowAction) OutputRegRange(regID int, rng Range) FlowBuilder {
	name := fmt.Sprintf("%s%d", NxmFieldReg, regID)
	return a.OutputFieldRange(name, rng)
}

// OutputInPort is an action to output packets to the ofport from where the packet enters the OFSwitch.
func (a *ofFlowAction) OutputInPort() FlowBuilder {
	outputAction := ofctrl.NewOutputInPort()
	a.builder.ofFlow.lastAction = outputAction
	return a.builder
}

// CT is an action to set conntrack marks and return CTAction to add actions that is executed with conntrack context.
func (a *ofFlowAction) CT(commit bool, tableID TableIDType, zone int) CTAction {
	base := ctBase{
		commit:  commit,
		force:   false,
		ctTable: uint8(tableID),
		ctZone:  uint16(zone),
	}
	var repr string
	if commit {
		repr += "commit"
	}
	if tableID != LastTableID {
		if repr != "" {
			repr += ","
		}
		repr += fmt.Sprintf("table=%d", tableID)
	}
	if zone > 0 {
		if repr != "" {
			repr += ","
		}
		repr += fmt.Sprintf("zone=%d", zone)
	}
	ct := &ofCTAction{
		ctBase:  base,
		builder: a.builder,
	}
	return ct
}

// ofCTAction is a struct to implement CTAction.
type ofCTAction struct {
	ctBase
	actions []openflow13.Action
	builder *ofFlowBuilder
}

// LoadToMark is an action to load data into ct_mark.
func (a *ofCTAction) LoadToMark(value uint32) CTAction {
	field, rng, _ := getFieldRange(NxmFieldCtMark)
	a.load(field, uint64(value), &rng)
	return a
}

// LoadToLabelRange is an action to load data into ct_label at specified range.
func (a *ofCTAction) LoadToLabelRange(value uint64, rng *Range) CTAction {
	field, _, _ := getFieldRange(NxmFieldCtLabel)
	a.load(field, value, rng)
	return a
}

func (a *ofCTAction) load(field *openflow13.MatchField, value uint64, rng *Range) {
	action := openflow13.NewNXActionRegLoad(rng.ToNXRange().ToOfsBits(), field, value)
	a.actions = append(a.actions, action)
}

// MoveToLabel is an action to move data into ct_mark.
func (a *ofCTAction) MoveToLabel(fromName string, fromRng, labelRng *Range) CTAction {
	fromField, _ := openflow13.FindFieldHeaderByName(fromName, false)
	toField, _ := openflow13.FindFieldHeaderByName(NxmFieldCtLabel, false)
	a.move(fromField, toField, uint16(fromRng.length()), uint16(fromRng[0]), uint16(labelRng[0]))
	return a
}

func (a *ofCTAction) move(fromField *openflow13.MatchField, toField *openflow13.MatchField, nBits, fromStart, toStart uint16) {
	action := openflow13.NewNXActionRegMove(nBits, fromStart, toStart, fromField, toField)
	a.actions = append(a.actions, action)
}

// CTDone sets the conntrack action in the Openflow rule and it returns FlowBuilder.
func (a *ofCTAction) CTDone() FlowBuilder {
	a.builder.Flow.ConnTrack(a.commit, a.force, &a.ctTable, &a.ctZone, a.actions...)
	return a.builder
}

// SetDstMAC is an action to modify packet destination MAC address to the specified address.
func (a *ofFlowAction) SetDstMAC(addr net.HardwareAddr) FlowBuilder {
	a.builder.SetMacDa(addr)
	return a.builder
}

// SetSrcMAC is an action to modify packet source MAC address to the specified address.
func (a *ofFlowAction) SetSrcMAC(addr net.HardwareAddr) FlowBuilder {
	a.builder.SetMacSa(addr)
	return a.builder
}

// SetARPSha is an action to modify ARP packet source hardware address to the specified address.
func (a *ofFlowAction) SetARPSha(addr net.HardwareAddr) FlowBuilder {
	a.builder.SetARPSha(addr)
	return a.builder
}

// SetARPTha is an action to modify ARP packet target hardware address to the specified address.
func (a *ofFlowAction) SetARPTha(addr net.HardwareAddr) FlowBuilder {
	a.builder.SetARPTha(addr)
	return a.builder
}

// SetARPSpa is an action to modify ARP packet source protocol address to the specified address.
func (a *ofFlowAction) SetARPSpa(addr net.IP) FlowBuilder {
	a.builder.SetARPSpa(addr)
	return a.builder
}

// SetARPTpa is an action to modify ARP packet target protocol address to the specified address.
func (a *ofFlowAction) SetARPTpa(addr net.IP) FlowBuilder {
	a.builder.SetARPTpa(addr)
	return a.builder
}

// SetSrcIP is an action to modify packet source IP address to the specified address.
func (a *ofFlowAction) SetSrcIP(addr net.IP) FlowBuilder {
	a.builder.SetIPField(addr, "Src")
	return a.builder
}

// SetDstIP is an action to modify packet destination IP address to the specified address.
func (a *ofFlowAction) SetDstIP(addr net.IP) FlowBuilder {
	a.builder.SetIPField(addr, "Dst")
	return a.builder
}

// SetTunnelDst is an action to modify packet tunnel destination address to the specified address.
func (a *ofFlowAction) SetTunnelDst(addr net.IP) FlowBuilder {
	a.builder.SetIPField(addr, "TunDst")
	return a.builder
}

// LoadARPOperation is an action to Load data to NXM_OF_ARP_OP field.
func (a *ofFlowAction) LoadARPOperation(value uint16) FlowBuilder {
	a.builder.ofFlow.LoadReg(NxmFieldARPOp, uint64(value), openflow13.NewNXRange(0, 15))
	return a.builder
}

// LoadRange is an action to Load data to the target field at specified range.
func (a *ofFlowAction) LoadRange(name string, value uint32, rng Range) FlowBuilder {
	a.builder.ofFlow.LoadReg(name, uint64(value), rng.ToNXRange())
	return a.builder
}

// LoadRegRange is an action to Load data to the target register at specified range.
func (a *ofFlowAction) LoadRegRange(regID int, value uint32, rng Range) FlowBuilder {
	name := fmt.Sprintf("%s%d", NxmFieldReg, regID)
	a.builder.ofFlow.LoadReg(name, uint64(value), rng.ToNXRange())
	return a.builder
}

// Move is an action to copy all data from "fromField" to "toField". Fields with name "fromField" and "fromField" should
// have the same data length, otherwise there will be error when realizing the flow on OFSwitch.
func (a *ofFlowAction) Move(fromField, toField string) FlowBuilder {
	_, fromRange, _ := getFieldRange(fromField)
	_, toRange, _ := getFieldRange(fromField)
	return a.MoveRange(fromField, toField, fromRange, toRange)
}

// MoveRange is an action to move data from "fromField" at "fromRange" to "toField" at "toRange".
func (a *ofFlowAction) MoveRange(fromField, toField string, fromRange, toRange Range) FlowBuilder {
	a.builder.ofFlow.MoveRegs(fromField, toField, fromRange.ToNXRange(), toRange.ToNXRange())
	return a.builder
}

// Resubmit is an action to resubmit packet to the specified table with the port as new in_port. If port is empty string,
// the in_port field is not changed.
func (a *ofFlowAction) Resubmit(ofPort uint16, table TableIDType) FlowBuilder {
	ofTableID := uint8(table)
	resubmit := ofctrl.NewResubmit(&ofPort, &ofTableID)
	a.builder.ofFlow.lastAction = resubmit
	return a.builder
}

func (a *ofFlowAction) ResubmitToTable(table TableIDType) FlowBuilder {
	ofTableID := uint8(table)
	resubmit := ofctrl.NewResubmit(nil, &ofTableID)
	a.builder.ofFlow.lastAction = resubmit
	return a.builder
}

// DecTTL is an action to decrease TTL. It is used in routing functions implemented by Openflow.
func (a *ofFlowAction) DecTTL() FlowBuilder {
	a.builder.ofFlow.DecTTL()
	return a.builder
}

// Normal is an action to leverage OVS fwd table to forwarding packets.
func (a *ofFlowAction) Normal() FlowBuilder {
	normalAction := ofctrl.NewOutputNormal()
	a.builder.ofFlow.lastAction = normalAction
	return a.builder
}

// Conjunction is an action to add new conjunction configuration to conjunctive match flow.
func (a *ofFlowAction) Conjunction(conjID uint32, clauseID uint8, nClause uint8) FlowBuilder {
	a.builder.ofFlow.AddConjunction(conjID, clauseID, nClause)
	return a.builder
}

func getFieldRange(name string) (*openflow13.MatchField, Range, error) {
	field, err := openflow13.FindFieldHeaderByName(name, false)
	if err != nil {
		return field, Range{0, 0}, err
	}
	return field, Range{0, uint32(field.Length)*8 - 1}, nil
}
