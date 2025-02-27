use crate::*;

pub(crate) enum TimerAction {
    Start {
        id: uuid::Uuid,
        engine: Engine,
        duration: Option<std::time::Duration>,
    },
    Stop {
        id: uuid::Uuid,
    },
    Cancel {
        id: uuid::Uuid,
    },
    Shutdown,
}

pub(crate) struct Timer {
    pub tx: std::sync::mpsc::Sender<TimerAction>,
    pub thread: Option<std::thread::JoinHandle<()>>,
}

#[cfg(not(target_family = "windows"))]
extern "C" fn cleanup_timer() {
    let mut timer = match unsafe { TIMER.lock() } {
        Ok(x) => x,
        Err(e) => e.into_inner(),
    };
    drop(timer.take());
}

static mut TIMER: std::sync::Mutex<Option<Timer>> = std::sync::Mutex::new(None);

impl Timer {
    pub(crate) fn tx() -> std::sync::mpsc::Sender<TimerAction> {
        let mut timer = match unsafe { TIMER.lock() } {
            Ok(x) => x,
            Err(e) => e.into_inner(),
        };

        let timer = &mut *timer;

        match timer {
            None => Timer::init(timer),
            Some(t) => t.tx.clone(),
        }
    }

    pub fn init(timer: &mut Option<Timer>) -> std::sync::mpsc::Sender<TimerAction> {
        let (tx, rx) = std::sync::mpsc::channel();
        let thread = std::thread::spawn(move || {
            let mut plugins = std::collections::BTreeMap::new();

            macro_rules! handle {
                ($x:expr) => {
                    match $x {
                        TimerAction::Start {
                            id,
                            engine,
                            duration,
                        } => {
                            let duration = duration.map(|x| std::time::Instant::now() + x);
                            plugins.insert(id, (engine, duration));
                        }
                        TimerAction::Stop { id } => {
                            plugins.remove(&id);
                        }
                        TimerAction::Cancel { id } => {
                            if let Some((engine, _)) = plugins.remove(&id) {
                                engine.increment_epoch();
                            }
                        }
                        TimerAction::Shutdown => {
                            for (_, (engine, _)) in plugins.iter() {
                                engine.increment_epoch();
                            }
                            return;
                        }
                    }
                };
            }

            loop {
                if plugins.is_empty() {
                    if let Ok(x) = rx.recv() {
                        handle!(x)
                    }
                }

                plugins = plugins
                    .into_iter()
                    .filter(|(_k, (engine, end))| {
                        if let Some(end) = end {
                            let now = std::time::Instant::now();
                            if end <= &now {
                                engine.increment_epoch();
                                return false;
                            }
                        }
                        true
                    })
                    .collect();

                for x in rx.try_iter() {
                    handle!(x)
                }
            }
        });
        *timer = Some(Timer {
            thread: Some(thread),
            tx: tx.clone(),
        });

        #[cfg(not(target_family = "windows"))]
        unsafe {
            libc::atexit(cleanup_timer);
        }

        tx
    }
}

impl Drop for Timer {
    fn drop(&mut self) {
        let _ = self.tx.send(TimerAction::Shutdown);
        if let Some(thread) = self.thread.take() {
            let _ = thread.join();
        }
    }
}
