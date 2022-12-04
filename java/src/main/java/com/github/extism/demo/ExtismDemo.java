package com.github.extism.demo;

import jdk.incubator.foreign.*;

import java.lang.management.MemoryUsage;
import java.nio.channels.FileChannel;
import java.nio.file.Files;
import java.nio.file.Path;

import com.github.extism.extism_h;

public class ExtismDemo {

    public static void main(String[] args) throws Exception {

        var wasmPath = Path.of(System.getProperty("wasmPath", "../wasm/code.wasm"));
        var funcName = System.getProperty("funcName", "count_vowels");
        var input = System.getProperty("input", "hello world");

        var output = Extism.executeFunction(wasmPath, funcName, input);

        System.out.println(output);

    }

    public static class Extism {

        static {
            System.loadLibrary("extism");
        }

        public static String executeFunction(Path wasmPath, String functionName, String input) throws Exception {

            Addressable context = null;
            int plugin = 0;

            try (var scope = ResourceScope.newConfinedScope()) {
                var scopedAllocator = SegmentAllocator.nativeAllocator(scope);
                var wasmBytesLength = Files.size(wasmPath);
                var mappedWasm = MemorySegment.mapFile(wasmPath, 0, wasmBytesLength, FileChannel.MapMode.READ_ONLY,
                        scope);
                context = extism_h.extism_context_new();
                plugin = extism_h.extism_plugin_new(context, mappedWasm.address(), wasmBytesLength, false);

                var funcNameAddr = scopedAllocator.allocateUtf8String(functionName);
                var inputAddr = scopedAllocator.allocateUtf8String(input);
                var result = extism_h.extism_plugin_call(context, plugin, funcNameAddr, inputAddr,
                        inputAddr.byteSize());

                var outputBytes = extism_h.extism_plugin_output_data(context, plugin);
                var output = outputBytes.getUtf8String(0);
                return output;
            } finally {
                if (context != null) {
                    extism_h.extism_plugin_free(context, plugin);
                    extism_h.extism_context_free(context);
                }
            }
        }
    }

}
