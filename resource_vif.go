/*
 * The MIT License (MIT)
 * Copyright (c) 2016 Maksym Borodin <borodin.maksym@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
 * documentation files (the "Software"), to deal in the Software without restriction, including without limitation
 * the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software,
 * and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all copies or substantial portions
 * of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO
 * THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
 * THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF
 * CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 */

package main

import (
	"github.com/amfranz/go-xen-api-client"
	"github.com/hashicorp/terraform/helper/schema"
	"fmt"
	"strconv"
)

const (
	vifSchemaNetworkNameLabel        = "network_name_label"
	vifSchemaNetworkUUID             = "network_uuid"
	vifSchemaMac                     = "mac"
	vifSchemaMacGenerated            = "mac_autogenerated"
	vifSchemaMtu                     = "mtu"
	vifSchemaDevice                  = "device"
)

func readVIFsFromSchema(c *Connection, s []interface{}) ([]*VIFDescriptor, error) {
	vifs := make([]*VIFDescriptor, 0, len(s))

	for _, schm := range s {
		data := schm.(map[string]interface{})

		network := &NetworkDescriptor{}
		if id, ok := data[vifSchemaNetworkNameLabel]; ok {
			network.Name = id.(string)
		}
		if id, ok := data[vifSchemaNetworkUUID]; ok {
			network.UUID = id.(string)
		}
		if err := network.Load(c); err != nil {
			return nil, err
		}
		mtu := data[vifSchemaMtu].(int)
		device := data[vifSchemaDevice].(int)
		mac_autogenerated := data[vifSchemaMacGenerated].(bool)
		var mac string
		if !mac_autogenerated {
			mac = data[vifSchemaMac].(string)
		}

		vif := &VIFDescriptor{
			Network: network,
			MAC: mac,
			IsAutogeneratedMAC: mac_autogenerated,
			DeviceOrder: device,
			MTU: mtu,
		}

		vifs = append(vifs, vif)
	}

	return vifs, nil
}

func createVIF(c *Connection, vif *VIFDescriptor) (*VIFDescriptor, error) {
	// FIXME: Should be available to add VIF to running VM with PV drivers installed
	// TODO: Check PV driver status
	if vif.VM.PowerState == xenAPI.VMPowerStateRunning {
		return nil, fmt.Errorf("VM %q(%q) is in running state!", vif.VM.Name, vif.VM.UUID)
	}

	if vif.DeviceOrder == 0 {
		vif.DeviceOrder = vif.VM.VIFCount
	}

	vifObject := xenAPI.VIFRecord{
		VM: vif.VM.VMRef,
		Network: vif.Network.NetworkRef,
		MTU: vif.MTU,
		MACAutogenerated: vif.IsAutogeneratedMAC,
		MAC: vif.MAC,
		Device: strconv.Itoa(vif.DeviceOrder),
	}

	vifRef, err := c.client.VIF.Create(c.session, vifObject)
	if err != nil {
		return nil, err
	}

	vif.VIFRef = vifRef
	vif.Query(c)

	err = c.client.VIF.Plug(c.session, vifRef)
	if err != nil {
		return nil, err
	}

	return vif, nil
}

func resourceVIF() *schema.Resource {
	return &schema.Resource{

		Schema: map[string]*schema.Schema{
			vifSchemaNetworkNameLabel: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			vifSchemaNetworkUUID: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			vifSchemaMac: &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			vifSchemaMacGenerated: &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			vifSchemaMtu: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			vifSchemaDevice: &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

/*
func resourceVIFRead(d *schema.ResourceData, m interface{}) error {
	c := m.(*Connection)


	d.Set(vifSchemaNetworkNameLabel, net.NameLabel)
	d.Set(vifSchemaNetworkUUID, net.UUID)
	d.Set(vifSchemaMac, vif.MAC)
	d.Set(vifSchemaMtu, vif.MTU)
	d.Set(vifSchemaMacGenerated, vif.MACAutogenerated)
	d.Set(vifSchemaDevice, strconv.Atoi(vif.Device))

	return nil
}
*/