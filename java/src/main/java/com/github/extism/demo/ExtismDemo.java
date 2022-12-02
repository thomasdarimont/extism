package com.github.extism.demo;

import jdk.incubator.foreign.*;

import java.lang.management.MemoryUsage;
import java.nio.channels.FileChannel;
import java.nio.file.Files;
import java.nio.file.Path;

import com.github.extism.extism_h;

public class ExtismDemo {
    
    public static void main(String[] args) throws Exception {
        

        var libaryPath = Path.of("../target/release/libextism.so").toFile().getAbsolutePath();
        System.load(libaryPath);

        var input = System.getProperty("input","hello world");
        var funcName = System.getProperty("funcName", "count_vowels");
        var wasmPath = Path.of(System.getProperty("wasmPath","../wasm/code.wasm"));

        try (var scope = ResourceScope.newConfinedScope()) {
            
            var wasmBytesLength = Files.size(wasmPath);
            var mappedWasm = MemorySegment.mapFile(wasmPath, 0, wasmBytesLength,FileChannel.MapMode.READ_ONLY, scope);

            var strAddr = MemorySegment.allocateNative(input.getBytes().length*2, scope);
            strAddr.setUtf8String(0, input);

            var context = extism_h.extism_context_new();
            var plugin = extism_h.extism_plugin_new(context, mappedWasm.address(), wasmBytesLength, false);

            var funcNameAddr = MemorySegment.allocateNative(input.getBytes().length*2, scope);
            funcNameAddr.setUtf8String(0, funcName);
            
            var result = extism_h.extism_plugin_call(context, plugin, funcNameAddr, strAddr, strAddr.byteSize());

            var outputBytes = extism_h.extism_plugin_output_data(context, plugin);

            var output = outputBytes.getUtf8String(0);

            System.out.println(output);

            extism_h.extism_plugin_free(context, plugin);
            extism_h.extism_context_free(context);
        }
    }

}
