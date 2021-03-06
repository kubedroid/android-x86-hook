/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 * Copyright 2018 Quamotion bvba
 *
 */

package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net"
	"os"

	"google.golang.org/grpc"

	vmSchema "kubevirt.io/kubevirt/pkg/api/v1"
	hooks "kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	"kubevirt.io/kubevirt/pkg/log"
	domainSchema "github.com/libvirt/libvirt-go-xml"
)

const baseBoardManufacturerAnnotation = "smbios.vm.kubevirt.io/baseBoardManufacturer"
const videoModelAnnotation = "video.vm.kubevirt.io/model"
const vgpuAnnotation = "video.vm.kubevirt.io/vgpu"
const eglHeadlessAnnotation = "graphics.vm.kubevirt.io/eglHeadless"
const qemuArgsAnnotation = "qemu.vm.kubevirt.io/args"
const hookName = "android-x86"

type infoServer struct{}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: hookName,
		Versions: []string{
			hooksV1alpha1.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			&hooksInfo.HookPoint{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha1Server struct{}

func (s v1alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info("Hook's OnDefineDomain callback method has been called")

	vmiJSON := params.GetVmi()
	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	annotations := vmiSpec.GetAnnotations()

	domainXML := params.GetDomainXML()
	domain := domainSchema.Domain{}
	err = xml.Unmarshal(domainXML, &domain)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	if baseBoardManufacturer, found := annotations[baseBoardManufacturerAnnotation]; !found {
		log.Log.Infof("The '%s' attribute was not provided. Not configuring the baseboard manufacturer", baseBoardManufacturerAnnotation)
	} else {
		log.Log.Infof("Configuring the baseboard manufacturer to be '%s'", baseBoardManufacturer)
		// Add a new /os/smbios node, with mode set to sysinfo
		domain.OS.SMBios = &domainSchema.DomainSMBios{Mode: "sysinfo"}

		// Add an empty /sysinfo node if required
		if domain.SysInfo == nil {
			domain.SysInfo = &domainSchema.DomainSysInfo{}
		}

		// Populate the sysinfo node with the required values.
		domain.SysInfo.Type = "smbios"
		baseBoard := domainSchema.DomainSysInfoBaseBoard {}
		baseBoard.Entry = append(baseBoard.Entry, domainSchema.DomainSysInfoEntry {
			Name:  "manufacturer",
			Value: baseBoardManufacturer,
			})

		domain.SysInfo.BaseBoard = append(domain.SysInfo.BaseBoard, baseBoard)
	}

	if videoModel, found := annotations[videoModelAnnotation]; !found {
		log.Log.Infof("The '%s' attribute was not provided. Not configuring the video model", videoModelAnnotation)
	} else {
		log.Log.Infof("Configuring the video model to be '%s'", videoModel)

		if len(domain.Devices.Videos) == 0 {
			domain.Devices.Videos = append(domain.Devices.Videos, domainSchema.DomainVideo{})
		}

		domain.Devices.Videos[0].Model.Type = videoModel
	}

	if _, found := annotations[eglHeadlessAnnotation]; !found {
		log.Log.Infof("The '%s' attribute was not provided. Not configuring the egl-headless graphics", eglHeadlessAnnotation)
	} else {
		log.Log.Infof("Configuring the egl headless graphics")

		eglHeadlessGraphics := domainSchema.DomainGraphic {
			EGLHeadless: &domainSchema.DomainGraphicEGLHeadless{
			},
		}
		domain.Devices.Graphics = append(domain.Devices.Graphics, eglHeadlessGraphics)
	}

	if vgpu, found := annotations[vgpuAnnotation]; !found {
		log.Log.Infof("The '%s' attribute was not provided. Not configuring a vGPU", vgpuAnnotation)
	} else {
		log.Log.Infof("Configuring a vGPU with UUID %s", vgpu)

		pci_domain := uint(0x0000)
		pci_bus := uint(0x00)
		pci_slot := uint(0x05)
		pci_function := uint(0x0)

		hostDev := domainSchema.DomainHostdev {
			Managed: "no",
			SubsysMDev : &domainSchema.DomainHostdevSubsysMDev {
				Model: "vfio-pci",
				Display: "on",
				Source: &domainSchema.DomainHostdevSubsysMDevSource {
					Address: &domainSchema.DomainAddressMDev {
						UUID: vgpu,
					},
				},
			},
			Address: &domainSchema.DomainAddress {
				PCI: &domainSchema.DomainAddressPCI {
					Domain: &pci_domain,
					Bus: &pci_bus,
					Slot: &pci_slot,
					Function: &pci_function,
				},
			},
		}
		domain.Devices.Hostdevs = append(domain.Devices.Hostdevs, hostDev)
	}

	if qemuArgsString, found := annotations[qemuArgsAnnotation]; !found {
		log.Log.Infof("The '%s' attribute was not provided. Not configuring additional qemu arguments", qemuArgsAnnotation)
	} else {
		log.Log.Infof("Configuring the qemu commands to be '%s'", qemuArgsString)

		var qemuArgs []string
		err := json.Unmarshal([]byte(qemuArgsString), &qemuArgs)
		if err != nil {
			log.Log.Reason(err).Errorf("Failed to unmarshal qemu arguments: '%s'. Ignoring the qemu arguments.", qemuArgsString)
		} else {
			if domain.QEMUCommandline == nil {
				domain.QEMUCommandline = &domainSchema.DomainQEMUCommandline{}
			}

			for _, qemuArg := range qemuArgs {
				log.Log.Infof("Adding the qemu argument '%s'", qemuArg)

				domain.QEMUCommandline.Args = append(domain.QEMUCommandline.Args, domainSchema.DomainQEMUCommandlineArg{ Value: qemuArg, })
			}
		}
	}

	newDomainXML, err := xml.Marshal(domain)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %s", err.Error())
		panic(err)
	}

	log.Log.Info("Successfully updated original domain spec with requested attributes")

	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func main() {
	// Start listening on /var/run/kubevirt-hooks/android-x86.sock,
	// and register an infoServer (to expose information about this
	// hook) and a callback server (which does the heavy lifting).
	log.InitializeLogging("android-x86-hook-sidecar")

	socketPath := hooks.HookSocketsSharedDirectory + "/" + hookName + ".sock"
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, infoServer{})
	hooksV1alpha1.RegisterCallbacksServer(server, v1alpha1Server{})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha1' services on socket %s", socketPath)
	server.Serve(socket)
}
