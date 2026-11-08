package io.github.jn0v.traceglass.feature.tracing

import io.github.jn0v.traceglass.core.session.SessionRepository
import io.github.jn0v.traceglass.core.timelapse.SnapshotStorage
import androidx.lifecycle.viewModelScope
import io.mockk.mockk
import kotlinx.coroutines.cancel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.jupiter.api.AfterEach
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Nested
import org.junit.jupiter.api.Test
import java.io.File

@OptIn(ExperimentalCoroutinesApi::class)
class TracingViewModelTest {

    private val testDispatcher = StandardTestDispatcher()

    @BeforeEach
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
    }

    @AfterEach
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun createViewModel(snapshotStorage: SnapshotStorage): TracingViewModel =
        TracingViewModel(
            flashlightController = mockk(relaxed = true),
            sessionRepository = FakeSessionRepository(),
            settingsRepository = mockk(relaxed = true),
            timelapseOps = TimelapseOperations(snapshotStorage, mockk(relaxed = true), mockk(relaxed = true), mockk(relaxed = true), File("/tmp")),
            ioDispatcher = testDispatcher,
            mainDispatcher = testDispatcher
        )

    @Nested
    inner class Timelapse {
        private val fakeStorage = FakeSnapshotStorage()

        @Test
        fun `startTimelapse sets recording state`() = runTest(testDispatcher) {
            val vm = createViewModel(snapshotStorage = fakeStorage)
            vm.startTimelapse()
            testDispatcher.scheduler.runCurrent()
            assertTrue(vm.uiState.value.isTimelapseRecording)
            assertFalse(vm.uiState.value.isTimelapsePaused)
            vm.viewModelScope.cancel()
        }
    }
}

private class FakeSnapshotStorage : SnapshotStorage {
    val savedSnapshots = mutableListOf<Pair<ByteArray, Int>>()
    var fakeSnapshotFiles: List<File> = emptyList()
    var fakeSnapshotCount: Int? = null
    var clearCalled: Boolean = false

    override fun saveSnapshot(jpegData: ByteArray, index: Int): File {
        savedSnapshots.add(jpegData to index)
        return File("/fake/snapshot_${"%04d".format(index)}.jpg")
    }

    override fun getSnapshotFiles(): List<File> = fakeSnapshotFiles
    override fun getSnapshotCount(): Int = fakeSnapshotCount ?: savedSnapshots.size
    override fun clear() {
        savedSnapshots.clear()
        fakeSnapshotCount = 0
        clearCalled = true
    }
}

