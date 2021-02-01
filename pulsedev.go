package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/jfreymuth/pulse/proto"
)

type PulseSource struct {
	Index  uint32
	Handle *os.File
}

type PulseSink struct {
	Index  uint32
	Handle *os.File
}

var quoter = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func quote(val string) string {
	return `"` + quoter.Replace(val) + `"`
}

func propList(kv ...string) string {
	out := ""
	for i := 0; i < len(kv)-1; i += 2 {
		if out != "" {
			out += " "
		}
		out += kv[i] + "=" + quote(kv[i+1])
	}
	return out
}

func createPipeSource(name, desc, icon string, latencyMs float64) (*PulseSource, error) {
	var err error
	var resp proto.LoadModuleReply
	var file *os.File

	bufferBits := int(48000 * 4 * 1 * latencyMs / 1000)

	tmpFile := "/tmp/nDAX-" + name + ".pipe"

	err = pc.RawRequest(
		&proto.LoadModule{
			Name: "module-pipe-source",
			Args: propList(
				"source_name", name,
				"file", tmpFile,
				"rate", "48000",
				"format", "float32be",
				"channels", "1",
				"source_properties", fmt.Sprintf("device.buffering.buffer_size=%d device.icon_name=%s device.description='%s'", bufferBits, icon, desc),
			),
		},
		&resp,
	)

	if err != nil {
		return nil, fmt.Errorf("load-module module-pipe-source: %w", err)
	}

	if file, err = os.OpenFile(tmpFile, os.O_RDWR, 0755); err != nil {
		destroyModule(resp.ModuleIndex)
		return nil, fmt.Errorf("OpenFile %s: %w", tmpFile, err)
	}

	return &PulseSource{
		Index:  resp.ModuleIndex,
		Handle: file,
	}, nil
}

func createPipeSink(name, desc, icon string) (*PulseSink, error) {
	var err error
	var resp proto.LoadModuleReply
	var file *os.File

	tmpFile := "/tmp/nDAX-" + name + ".pipe"

	err = pc.RawRequest(
		&proto.LoadModule{
			Name: "module-pipe-sink",
			Args: propList(
				"sink_name", name,
				"file", tmpFile,
				"rate", "48000",
				"format", "float32be",
				"channels", "1",
				"use_system_clock_for_timing", "yes",
				"sink_properties", fmt.Sprintf("device.icon_name=%s device.description='%s'", icon, desc),
			),
		},
		&resp,
	)

	if err != nil {
		return nil, fmt.Errorf("load-module module-pipe-sink: %w", err)
	}

	if file, err = os.OpenFile(tmpFile, os.O_RDONLY, 0755); err != nil {
		destroyModule(resp.ModuleIndex)
		return nil, fmt.Errorf("OpenFile %s: %w", tmpFile, err)
	}

	return &PulseSink{
		Index:  resp.ModuleIndex,
		Handle: file,
	}, nil
}

func destroyModule(index uint32) error {
	err := pc.RawRequest(
		&proto.UnloadModule{
			ModuleIndex: index,
		},
		nil,
	)

	return err
}

func (s *PulseSource) Close() {
	if s.Handle != nil {
		s.Handle.Close()
	}
	destroyModule(s.Index)
}

func (s *PulseSink) Close() {
	if s.Handle != nil {
		s.Handle.Close()
	}
	destroyModule(s.Index)
}

func getModules() ([]*proto.GetModuleInfoReply, error) {
	var ret proto.GetModuleInfoListReply
	err := pc.RawRequest(
		&proto.GetModuleInfoList{},
		&ret,
	)
	if err != nil {
		return nil, err
	} else {
		return []*proto.GetModuleInfoReply(ret), nil
	}
}
