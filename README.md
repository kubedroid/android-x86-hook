# android-x86-hook

[![Docker Repository on Quay](https://quay.io/repository/kubedroid/android-x86-hook/status "Docker Repository on Quay")](https://quay.io/repository/kubedroid/android-x86-hook)

The android-x86-hook container implements a custom KubeVirt hook, allowing you to further customize your Android-x86
VM.

## Using this hook

To use this hook, add a `hooks.kubevirt.io/hookSidecars` annotation to your VMI, and specify the `quay.io/kubedroid/android-x86-hook:latest`
docker image for the hook.

## Supported annotations

This hook currently supports these annotations:

| Annotation                                   | Description                                                                                                       | Sample value                           |
|----------------------------------------------|-------------------------------------------------------------------------------------------------------------------|----------------------------------------|
| `smbios.vm.kubevirt.io/baseBoardManufacturer`| Allows you to set the name of the manufacturer of the base board inside the VMI.                                  | `Quamotion`                            |
| `video.vm.kubevirt.io/model`                 | Allows you to specify the model of the graphics card inside the VMI. Set to none to disable the default graphics. | `none`                                 | 
| `video.vm.kubevirt.io/vgpu`                  | Attaches a vGPU to the VMI as a vfio-pci device. Specify the UUID of the vGPU.                                    | `a297db4a-f4c2-11e6-90f6-d3b88d6c9525` |
| `graphics.vm.kubevirt.io/eglHeadless`        | Enables headless egl for a VMI.                                                                                   | `yes`                                  |
| `qemu.vm.kubevirt.io/args`                   | Allows you to pass arbitrary arguments to qemu. You must format them as a JSON array                              | ` { "--usb-device-tablet "}`           |

## Example

In this example, we'll create a VMI (a VM running inside KubeVirt) running Fedora which has access
to a vGPU.

You'll need to have the Intel GPU and Intel vGPU device plugin configured on your Kubernetes cluster,
and you need to have a configured Intel vGPU. For the purposes of this example, we're assuming the
UUID of your vGPU is `a297db4a-f4c2-11e6-90f6-d3b88d6c9525`

To use this hook to add a vGPU to your VMI:

```yaml
apiVersion: kubevirt.io/v1alpha2
kind: VirtualMachineInstance
metadata:
  annotations:
    hooks.kubevirt.io/hookSidecars: '[{"image": "quay.io/kubedroid/android-x86-hook:latest"}]'
    video.vm.kubevirt.io/vgpu: "a297db4a-f4c2-11e6-90f6-d3b88d6c9525"
    graphics.vm.kubevirt.io/eglHeadless: "true"
    video.vm.kubevirt.io/model: "none"
spec:
  volumes:
    - name: containervolume
      containerDisk:
        image: kubevirt/fedora-cloud-registry-disk-demo:v0.10.0
  domain:
    resources:
      # This will request an Intel vGPU for the VMI.
      # It requires the Intel GPU device plugin, see
      # https://github.com/kubedroid/intel-device-plugins-for-kubernetes/tree/vgpu
      limits:
        gpu.intel.com/i915-v: 1
        gpu.intel.com/i915: 1
```

## About

This repository is part of [KubeDroid](https://github.com/kubedroid). You can use KubeDroid to run Android-x86
emulators inside Kubernetes clusters, using [KubeVirt](https://kubevirt.io)

KubeDroid is sponsored by [Quamotion](http://quamotion.mobi). Quamotion provides test automation software for
mobile devices.
