package io.github.jn0v.traceglass.feature.tracing

import io.github.jn0v.traceglass.core.session.SessionRepository
import io.github.jn0v.traceglass.core.timelapse.TimelapseSession
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update

class TracingViewModel(
    private val flashlightController: FlashlightController,
    private val transformCalculator: OverlayTransformCalculator = OverlayTransformCalculator(),
    private val trackingStateManager: TrackingStateManager = TrackingStateManager(),
    private val sessionRepository: SessionRepository,
    private val settingsRepository: SettingsRepository,
    private val cameraManager: CameraManager? = null,
    internal val frameAnalyzer: FrameAnalyzer? = null,
    private val timelapseOps: TimelapseOperations? = null,
    private val ioDispatcher: kotlinx.coroutines.CoroutineDispatcher = Dispatchers.IO,
    private val mainDispatcher: kotlinx.coroutines.CoroutineDispatcher = Dispatchers.Main
) : ViewModel() {

    // Convenience accessors for grouped timelapse dependencies
    private val snapshotStorage get() = timelapseOps?.snapshotStorage
    private val timelapseCompiler get() = timelapseOps?.compiler
    private val videoExporter get() = timelapseOps?.exporter
    private val videoSharer get() = timelapseOps?.sharer
    private val cacheDir get() = timelapseOps?.cacheDir

    private val _uiState = MutableStateFlow(
        TracingUiState(hasFlashlight = flashlightController.hasFlashlight)
    )
    val uiState: StateFlow<TracingUiState> = _uiState.asStateFlow()

    // Timelapse state
    private var timelapseSession: TimelapseSession? = null
    private var timelapseObserverJob: Job? = null

    fun startTimelapse() {
        val storage = snapshotStorage ?: return
        val analyzer = frameAnalyzer ?: return

        // Resume existing paused session instead of creating a new one
        val existing = timelapseSession
        if (existing != null) {
            if (_uiState.value.isTimelapsePaused) {
                existing.resume(viewModelScope)
            }
            return
        }

        val session = TimelapseSession(onCapture = { index ->
            analyzer.snapshotCallback = { bytes ->
                storage.saveSnapshot(bytes, index)
            }
        })
        timelapseSession = session
        session.start(viewModelScope)
        observeTimelapseState(session)
    }

    fun pauseTimelapse() {
        val session = timelapseSession ?: return
        session.pause()
        frameAnalyzer?.snapshotCallback = null
    }

    fun resumeTimelapse() {
        val session = timelapseSession ?: return
        session.resume(viewModelScope)
    }

    fun stopTimelapse() {
        val session = timelapseSession ?: return
        session.stop()
        frameAnalyzer?.snapshotCallback = null
        timelapseObserverJob?.cancel()
        timelapseObserverJob = null
        timelapseSession = null

        _uiState.update {
            it.copy(
                isTimelapseRecording = false,
                isTimelapsePaused = false,
                snapshotCount = 0
            )
        }

        viewModelScope.launch(ioDispatcher) {
            val snapshotFiles = snapshotStorage?.getSnapshotFiles() ?: emptyList()
            if (snapshotFiles.isNotEmpty() && timelapseCompiler != null && cacheDir != null) {
                compileTimelapse(snapshotFiles)
            }
        }
    }

    fun onDismissPostCompilationDialog() {
        _uiState.update { it.copy(showPostCompilationDialog = false) }
    }

    private fun cleanupTempFiles() {
        snapshotStorage?.clear()
    }
}
