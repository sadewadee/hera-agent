---
name: keras_training
description: "Model training with Keras and TensorFlow"
version: "1.0"
trigger: "keras tensorflow training model fit"
platforms: []
requires_tools: ["run_command"]
---

# Keras Training

## Purpose
Train deep learning models using Keras with TensorFlow backend, leveraging built-in callbacks, data pipelines, and distributed training.

## Instructions
1. Define model architecture with Sequential or Functional API
2. Compile with optimizer, loss, and metrics
3. Prepare data with tf.data pipelines
4. Train with model.fit() and appropriate callbacks
5. Evaluate and export the trained model

## Model Definition
```python
import tensorflow as tf

model = tf.keras.Sequential([
    tf.keras.layers.Dense(256, activation="relu", input_shape=(768,)),
    tf.keras.layers.Dropout(0.3),
    tf.keras.layers.Dense(128, activation="relu"),
    tf.keras.layers.Dropout(0.2),
    tf.keras.layers.Dense(10, activation="softmax"),
])

model.compile(
    optimizer=tf.keras.optimizers.Adam(learning_rate=1e-3),
    loss="sparse_categorical_crossentropy",
    metrics=["accuracy"],
)
```

## tf.data Pipeline
```python
dataset = tf.data.Dataset.from_tensor_slices((features, labels))
dataset = (dataset
    .shuffle(10000)
    .batch(32)
    .prefetch(tf.data.AUTOTUNE)
)
```

## Callbacks
```python
callbacks = [
    tf.keras.callbacks.ModelCheckpoint("best_model.keras", save_best_only=True),
    tf.keras.callbacks.EarlyStopping(patience=5, restore_best_weights=True),
    tf.keras.callbacks.ReduceLROnPlateau(factor=0.5, patience=3),
    tf.keras.callbacks.TensorBoard(log_dir="./logs"),
]

model.fit(train_dataset, validation_data=val_dataset, epochs=50, callbacks=callbacks)
```

## Distributed Training
```python
strategy = tf.distribute.MirroredStrategy()
with strategy.scope():
    model = create_model()
    model.compile(optimizer="adam", loss="sparse_categorical_crossentropy")
model.fit(train_dataset, epochs=10)
```

## Best Practices
- Use tf.data pipelines with prefetch for GPU utilization
- Enable mixed precision with `tf.keras.mixed_precision.set_global_policy("mixed_float16")`
- Use model.summary() to verify architecture before training
- Save with model.save("model.keras") for Keras format (recommended over H5)
- Profile with TensorBoard for bottleneck identification
