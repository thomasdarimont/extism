package org.extism.sdk;

import org.extism.sdk.manifest.ManifestWasmData;
import org.extism.sdk.manifest.ManifestWasm;
import org.extism.sdk.manifest.Manifest;

import com.sun.jna.Pointer;

// ExtismContext is used to store and manage plugins
public class Context {

    // A pointer to the ExtismContext struct
    private Pointer contextPointer;

    /**
     * Create a new Context. A Context is used to manage Plugins
     * and make sure they are cleaned up when you are done with them.
     */
    public Context() {
        this.contextPointer = LibExtism.INSTANCE.extism_context_new();
    }

    /**
     * Create a new plugin managed by this context.
     *
     * @param manifest The manifest for the plugin
     * @param withWASI Set to true to enable WASI
     * @return
     */
    public Plugin newPlugin(Manifest manifest, boolean withWASI) {
        return new Plugin(this, manifest, withWASI);
    }

    /**
     * Frees the context *and* frees all its Plugins. Use {@link #reset()}, if you just want to
     * free the plugins but keep the context. You should ensure this is called when you are done
     * with the context.
     */
    public void free() {
        LibExtism.INSTANCE.extism_context_free(this.contextPointer);
    }

    /**
     * Resets the context by freeing all its Plugins. Unlike {@link #free()}, it does not
     * free the context itself.
     */
    public void reset() {
        LibExtism.INSTANCE.extism_context_reset(this.contextPointer);
    }

    /**
     * Get the version string of the linked Extism Runtime.
     * 
     * @return
     */
    public String getVersion() {
        return LibExtism.INSTANCE.extism_version();
    }

    // Get the error associated with a context, if plugin is null then the context
    // error will be returned
    protected String error(Plugin plugin) {
        return LibExtism.INSTANCE.extism_error(this.contextPointer, plugin == null ? -1 : plugin.getIndex());
    }

    protected Pointer getPointer() {
        return this.contextPointer;
    }

}
