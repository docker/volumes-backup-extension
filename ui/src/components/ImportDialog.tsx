import React, { useContext, useState } from "react";
import {
  Alert,
  Button,
  CircularProgress,
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

import { useCreateVolume } from "../hooks/useCreateVolume";
import { useImportFromPath } from "../hooks/useImportFromPath";
import { ImageAutocomplete } from "./ImageAutocomplete";
import { useImportFromImage } from "../hooks/useImportFromImage";
import { MyContext } from "..";
import { VolumeOrInput } from "./VolumeOrInput";
import { RegistryImageInput } from "./RegistryImageInput";
import { usePullFromRegistry } from "../hooks/usePullFromRegistry";

const ddClient = createDockerDesktopClient();

interface Props {
  open: boolean;
  onClose(v: boolean): void;
  volumes: unknown[];
}

export default function ImportDialog({ volumes, open, onClose }: Props) {
  const [fromRadioValue, setFromRadioValue] = useState<
    "file" | "image" | "pull-registry"
  >("file");
  const [image, setImage] = useState<string>("");
  const [volumeName, setVolumeName] = useState("");
  const [volumeHasError, setVolumeHasError] = useState(false);
  const [path, setPath] = useState<string>("");
  const [registryImage, setRegistryImage] = useState("");
  const [registryImageError, setRegistryImageError] = useState("");

  // when executed from a Volume context we don't need to create it.
  const context = useContext(MyContext);
  const selectedVolumeName = context.store.volume?.volumeName;

  const { createVolume, isInProgress: isCreating } = useCreateVolume();
  const { importVolume, isInProgress: isImportingFromPath } =
    useImportFromPath();
  const { loadImage, isInProgress: isImportingFromImage } =
    useImportFromImage();
  const { pullFromRegistry, isLoading: isPulling } = usePullFromRegistry();

  const selectImportTarGzFile = () => {
    ddClient.desktopUI.dialog
      .showOpenDialog({
        properties: ["openFile"],
        filters: [{ name: ".tar.gz", extensions: ["tar.gz"] }], // should contain extension without wildcards or dots
      })
      .then((result) => {
        if (result.canceled) {
          return;
        }

        setPath(result.filePaths[0]);
      });
  };

  const handleCreateVolume = async () => {
    if (selectedVolumeName) return;
    return await createVolume(volumeName);
  };

  const createAndImport = async () => {
    const volumeId = await handleCreateVolume();
    onClose(true);

    if (fromRadioValue === "file") {
      await importVolume({
        volumeName: volumeId?.[0] || selectedVolumeName,
        path,
      });
    } else if (fromRadioValue === "image") {
      await loadImage({
        volumeName: volumeId?.[0] || selectedVolumeName,
        imageName: image,
      });
    } else {
      await pullFromRegistry({
        imageName: registryImage,
        volumeId: volumeId?.[0],
      });
    }
  };

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setFromRadioValue(
      (event.target as HTMLInputElement).value as "file" | "image"
    );
  };

  const renderFormControlFile = () => {
    return (
      <>
        <FormControlLabel value="file" control={<Radio />} label="Local file" />
        <Stack pt={1} pb={2} pl={4}>
          <Typography pb={1} variant="body2">
            Select a file (.tar.gz) whose content is to be imported into the{" "}
            {selectedVolumeName ? "existing" : "new"} volume.
          </Typography>
          {fromRadioValue === "file" && (
            <Grid container alignItems="center" gap={2}>
              <Grid item flex={1}>
                <TextField
                  fullWidth
                  disabled
                  label={path ? "" : "File name"}
                  value={path}
                  onClick={selectImportTarGzFile}
                />
              </Grid>
              <Button
                size="large"
                variant="outlined"
                onClick={selectImportTarGzFile}
              >
                Select file
              </Button>
            </Grid>
          )}
        </Stack>
      </>
    );
  };

  const renderImageRadioButton = () => {
    return (
      <>
        <FormControlLabel
          value="image"
          control={<Radio />}
          label="Local image"
        />
        <Stack pt={1} pb={2} pl={4} width="100%">
          <Typography pb={1} variant="body2">
            Select an image whose content is to be imported into the new volume.
          </Typography>
          {fromRadioValue === "image" && (
            <ImageAutocomplete
              value={image}
              onChange={(v) => setImage(v as any)}
            />
          )}
        </Stack>
      </>
    );
  };

  const renderPullFromRegistryRadioButton = () => {
    return (
      <>
        <FormControlLabel
          value="pull-registry"
          control={<Radio />}
          label="Registry"
        />
        <Stack pt={1} pb={2} pl={4} width="100%">
          <Typography pb={2} variant="body2">
            Pull content from a registry like DockerHub or GitHub Container
            Registry.
          </Typography>
          {fromRadioValue === "pull-registry" && (
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
    <Dialog fullWidth maxWidth="sm" open={open} onClose={onClose}>
      <DialogTitle>
        {selectedVolumeName ? "Import content" : "Import into a new volume"}
      </DialogTitle>
      <DialogContent>
        <Stack>
          {selectedVolumeName && (
            <Alert
              sx={(theme) => ({ marginBottom: theme.spacing(2) })}
              severity="warning"
            >
              Any existing data inside the volume will be replaced.
            </Alert>
          )}
          <FormControl>
            <FormLabel id="from-label">
              <Typography variant="h3" mb={1}>
                From:
              </Typography>
            </FormLabel>
            <RadioGroup
              aria-labelledby="from-label"
              defaultValue="file"
              name="radio-buttons-group"
              value={fromRadioValue}
              onChange={handleChange}
            >
              {renderFormControlFile()}
              {renderImageRadioButton()}
              {renderPullFromRegistryRadioButton()}
            </RadioGroup>
          </FormControl>

          <FormControl>
            <FormLabel id="to-label">
              <Typography variant="h3" mt={3} mb={1}>
                To:
              </Typography>
            </FormLabel>
            <VolumeOrInput
              value={volumeName}
              hasError={volumeHasError}
              setHasError={setVolumeHasError}
              onChange={setVolumeName}
              volumes={volumes}
            />
          </FormControl>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button
          variant="outlined"
          onClick={() => {
            setPath("");
            onClose(false);
          }}
        >
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={createAndImport}
          disabled={Boolean(
            (fromRadioValue === "file" && !path) ||
              (fromRadioValue === "pull-registry" &&
                (!registryImage || Boolean(registryImageError))) ||
              (fromRadioValue === "image" && !image) ||
              (volumeName && volumeHasError)
          )}
        >
          Import
        </Button>
      </DialogActions>
    </Dialog>
  );
}
