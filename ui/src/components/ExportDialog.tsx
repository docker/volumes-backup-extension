import React, { useContext, useState } from "react";
import {
  Alert,
  Button,
  FormControl,
  FormControlLabel,
  FormLabel,
  Grid,
  Radio,
  RadioGroup,
  Stack,
  TextField,
  Typography,
} from "@mui/material";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import { createDockerDesktopClient } from "@docker/extension-api-client";

import { MyContext } from "../index";
import { useExportVolume } from "../hooks/useExportVolume";
import { VolumeOrInput } from "./VolumeOrInput";
import { ImageAutocomplete } from "./ImageAutocomplete";
import { useExportToImage } from "../hooks/useExportToImage";
import { NewImageInput } from "./NewImageInput";
import { usePushVolumeToRegistry } from "../hooks/usePushVolumeToRegistry";
import { RegistryImageInput } from "./RegistryImageInput";
import { track } from "../common/track";

const ddClient = createDockerDesktopClient();

interface Props {
  open: boolean;
  onClose(v: boolean): void;
}

export default function ExportDialog({ open, onClose }: Props) {
  const context = useContext(MyContext);

  const [fromRadioValue, setFromRadioValue] = useState<
    "directory" | "local-image" | "new-image" | "push-registry"
  >("directory");
  const [fileName, setFileName] = useState<string>(
    `${context.store.volume.volumeName}.tar.gz`
  );
  const [path, setPath] = useState<string>("");
  const [image, setImage] = useState<string>("");
  const [newImage, setNewImage] = useState<string>("");
  const [newImageHasError, setNewImageHasError] = useState<boolean>(false);
  const [registryImage, setRegistryImage] = useState("");
  const [registryImageError, setRegistryImageError] = useState("");

  const { isLoading: isExportingToFile, exportVolume } = useExportVolume();
  const { isLoading: isExportingToImage, exportToImage } = useExportToImage();
  const { isLoading: isPushingToRegistry, pushVolumeToRegistry } =
    usePushVolumeToRegistry();
  const selectExportDirectory = () => {
    ddClient.desktopUI.dialog
      .showOpenDialog({
        properties: ["openDirectory"],
      })
      .then((result) => {
        if (result.canceled) {
          return;
        }

        setPath(result.filePaths[0]);
      });
  };

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setFromRadioValue(
      (event.target as HTMLInputElement).value as
        | "directory"
        | "local-image"
        | "new-image"
    );
  };

  const handleExport = () => {
    track({
      action: "ExportVolume",
      volumeSize: context.store.volume.volumeBytes,
    });
    if (fromRadioValue === "directory") {
      exportVolume({ path, fileName });
    } else if (fromRadioValue === "new-image") {
      exportToImage({ imageName: newImage });
    } else if (fromRadioValue === "local-image") {
      exportToImage({ imageName: image });
    } else if (fromRadioValue === "push-registry") {
      pushVolumeToRegistry({ imageName: registryImage });
    }
    onClose(true);
  };

  const renderDirectoryRadioButton = () => {
    return (
      <>
        <FormControlLabel
          value="directory"
          control={<Radio />}
          label="Local file"
        />

        <Stack pt={1} pb={2} pl={4}>
          <Typography pb={1} variant="body2">
            Create a compressed file (gzip’ed tarball) in a selected directory
            with the content of a chosen volume.
          </Typography>
          {fromRadioValue === "directory" && (
            <>
              <TextField
                autoFocus
                margin="dense"
                id="file-name"
                label="File name"
                fullWidth
                defaultValue={`${context.store.volume.volumeName}.tar.gz`}
                spellCheck={false}
                onChange={(e) => {
                  setFileName(e.target.value);
                }}
              />
              <Grid container alignItems="center" gap={2}>
                <Grid item flex={1}>
                  <TextField
                    fullWidth
                    disabled
                    margin="dense"
                    id="directory"
                    label={path ? "" : "Directory"}
                    value={path}
                    onClick={selectExportDirectory}
                  />
                </Grid>
                <Button
                  size="large"
                  variant="outlined"
                  onClick={selectExportDirectory}
                >
                  Select directory
                </Button>
              </Grid>
            </>
          )}
        </Stack>
      </>
    );
  };

  const renderLocalImageRadioButton = () => {
    return (
      <>
        <FormControlLabel
          value="local-image"
          control={<Radio />}
          label="Local image"
        />
        <Stack pt={1} pb={2} pl={4} width="100%">
          <Typography pb={1} variant="body2">
            Copy the volume content to a busybox image in the /volume-data
            directory.
          </Typography>
          {fromRadioValue === "local-image" && (
            <ImageAutocomplete
              value={image}
              onChange={(v) => setImage(v as any)}
            />
          )}
        </Stack>
      </>
    );
  };

  const renderNewImageRadioButton = () => {
    return (
      <>
        <FormControlLabel
          value="new-image"
          control={<Radio />}
          label="New image"
        />
        <Stack pt={1} pb={2} pl={4} width="100%">
          <Typography pb={1} variant="body2">
            Create a new image and copy the volume’s content into it.
          </Typography>
          {fromRadioValue === "new-image" && (
            <NewImageInput
              value={newImage}
              onChange={setNewImage}
              hasError={newImageHasError}
              setHasError={setNewImageHasError}
            />
          )}
        </Stack>
      </>
    );
  };

  const renderPushToRegistryRadioButton = () => {
    return (
      <>
        <FormControlLabel
          value="push-registry"
          control={<Radio />}
          label="Registry"
        />
        <Stack pt={1} pb={2} pl={4} width="100%">
          <Typography pb={2} variant="body2">
            Push the volume content to a registry like DockerHub or GitHub
            Container Registry.
          </Typography>
          {fromRadioValue === "push-registry" && (
            <RegistryImageInput
              error={registryImageError}
              value={registryImage}
              setValue={setRegistryImage}
              setError={setRegistryImageError}
            />
          )}
        </Stack>
      </>
    );
  };

  return (
    <Dialog open={open} onClose={() => onClose(false)}>
      <DialogTitle>Export content</DialogTitle>
      <DialogContent>
        <FormControl>
          <FormLabel id="from-label">
            <Typography variant="h3" my={1}>
              From:
            </Typography>
          </FormLabel>
          <VolumeOrInput />
        </FormControl>
        <FormControl sx={{ width: "100%" }}>
          <FormLabel id="to-label">
            <Typography variant="h3" mt={3} mb={1}>
              To:
            </Typography>
          </FormLabel>
          {fromRadioValue === "local-image" && (
            <Alert
              sx={(theme) => ({ marginBottom: theme.spacing(2) })}
              severity="warning"
            >
              Any existing data inside /volume-data directory will be replaced.
            </Alert>
          )}
          <RadioGroup
            aria-labelledby="to-label"
            defaultValue="directory"
            name="radio-buttons-group"
            value={fromRadioValue}
            onChange={handleChange}
          >
            {renderDirectoryRadioButton()}
            {renderLocalImageRadioButton()}
            {renderNewImageRadioButton()}
            {renderPushToRegistryRadioButton()}
          </RadioGroup>
        </FormControl>
      </DialogContent>
      <DialogActions>
        <Button
          variant="outlined"
          onClick={() => {
            track({ action: "ExportVolumeCancel" });
            onClose(false);
          }}
        >
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={handleExport}
          disabled={
            (fromRadioValue === "directory" &&
              (path === "" || fileName === "")) ||
            (fromRadioValue === "local-image" && !image) ||
            (fromRadioValue === "new-image" &&
              (!newImage || newImageHasError)) ||
            (fromRadioValue === "push-registry" &&
              (!registryImage || Boolean(registryImageError)))
          }
        >
          Export
        </Button>
      </DialogActions>
    </Dialog>
  );
}
